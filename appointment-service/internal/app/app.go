package app

import (
	"appointment-service/internal/cache"
	"appointment-service/internal/client"
	"appointment-service/internal/event"
	"appointment-service/internal/middleware"
	"appointment-service/internal/repository"
	transportgrpc "appointment-service/internal/transport/grpc"
	"appointment-service/internal/usecase"
	pb "appointment-service/proto"
	"context"
	"database/sql"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// Run initialises all infrastructure, runs migrations, then starts the gRPC server.
func Run(port string, doctorServiceAddr string) {
	// ── Database ────────────────────────────────────────────────────────────
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("appointment-service: failed to open DB connection: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("appointment-service: failed to connect to database: %v", err)
	}
	log.Println("appointment-service: connected to PostgreSQL")

	// ── Migrations ──────────────────────────────────────────────────────────
	m, err := migrate.New("file://migrations", dbURL)
	if err != nil {
		log.Fatalf("appointment-service: failed to initialise migrations: %v", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("appointment-service: failed to apply migrations: %v", err)
	}
	log.Println("appointment-service: database migrations up-to-date")

	// ── Doctor Service gRPC client ──────────────────────────────────────────
	doctorClient, err := client.NewGRPCDoctorClient(doctorServiceAddr)
	if err != nil {
		log.Fatalf("appointment-service: failed to connect to Doctor Service at %s: %v", doctorServiceAddr, err)
	}

	// ── Redis (best-effort) ──────────────────────────────────────────────────
	var cacheStore cache.Cache
	var redisClient *redis.Client

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		log.Println("appointment-service: REDIS_URL not set — caching and rate limiting disabled")
		cacheStore = &cache.NoOpCache{}
	} else {
		opt, parseErr := redis.ParseURL(redisURL)
		if parseErr != nil {
			log.Printf("appointment-service: WARNING: invalid REDIS_URL %q: %v — caching disabled", redisURL, parseErr)
			cacheStore = &cache.NoOpCache{}
		} else {
			rdb := redis.NewClient(opt)
			pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if pingErr := rdb.Ping(pingCtx).Err(); pingErr != nil {
				log.Printf("appointment-service: WARNING: cannot reach Redis at %s: %v — caching disabled", redisURL, pingErr)
				cacheStore = &cache.NoOpCache{}
			} else {
				log.Printf("appointment-service: connected to Redis at %s", redisURL)
				redisClient = rdb
				cacheStore = cache.NewRedisCache(rdb)
			}
		}
	}

	// Cache TTL from environment.
	cacheTTL := 60 * time.Second
	if v := os.Getenv("CACHE_TTL_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cacheTTL = time.Duration(n) * time.Second
		}
	}

	// ── Message broker (best-effort) ────────────────────────────────────────
	var publisher event.Publisher
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		log.Println("appointment-service: NATS_URL not set — events will not be published")
		publisher = &event.NoOpPublisher{}
	} else {
		p, err := event.NewNATSPublisher(natsURL)
		if err != nil {
			log.Printf("appointment-service: WARNING: cannot connect to NATS at %s: %v — continuing without event publishing", natsURL, err)
			publisher = &event.NoOpPublisher{}
		} else {
			publisher = p
			log.Printf("appointment-service: connected to NATS at %s", natsURL)
		}
	}

	// ── Wire up layers ──────────────────────────────────────────────────────
	pgRepo := repository.NewPostgresAppointmentRepository(db)
	cachedRepo := cache.NewCachedAppointmentRepository(pgRepo, cacheStore, cacheTTL)
	uc := usecase.NewAppointmentUseCase(cachedRepo, doctorClient, publisher)
	server := transportgrpc.NewAppointmentServer(uc)

	// ── gRPC server options ──────────────────────────────────────────────────
	var serverOpts []grpc.ServerOption
	if redisClient != nil {
		rl := middleware.NewRateLimiter(redisClient)
		serverOpts = append(serverOpts, grpc.UnaryInterceptor(rl.UnaryServerInterceptor()))
		log.Println("appointment-service: rate limiter interceptor enabled")
	} else {
		log.Println("appointment-service: rate limiter disabled (Redis unavailable)")
	}

	// ── gRPC server ─────────────────────────────────────────────────────────
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("appointment-service: failed to listen on port %s: %v", port, err)
	}

	grpcServer := grpc.NewServer(serverOpts...)
	pb.RegisterAppointmentServiceServer(grpcServer, server)
	reflection.Register(grpcServer)

	log.Printf("Appointment Service gRPC listening on :%s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("appointment-service: failed to serve: %v", err)
	}
}
