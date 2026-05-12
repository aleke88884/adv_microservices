package app

import (
	"context"
	"database/sql"
	"doctor-service/internal/cache"
	"doctor-service/internal/event"
	"doctor-service/internal/middleware"
	"doctor-service/internal/repository"
	transportgrpc "doctor-service/internal/transport/grpc"
	"doctor-service/internal/usecase"
	pb "doctor-service/proto"
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
func Run(port string) {
	// ── Database ────────────────────────────────────────────────────────────
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("doctor-service: failed to open DB connection: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("doctor-service: failed to connect to database: %v", err)
	}
	log.Println("doctor-service: connected to PostgreSQL")

	// ── Migrations ──────────────────────────────────────────────────────────
	m, err := migrate.New("file://migrations", dbURL)
	if err != nil {
		log.Fatalf("doctor-service: failed to initialise migrations: %v", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("doctor-service: failed to apply migrations: %v", err)
	}
	log.Println("doctor-service: database migrations up-to-date")

	// ── Redis (best-effort) ──────────────────────────────────────────────────
	var cacheStore cache.Cache
	var redisClient *redis.Client

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		log.Println("doctor-service: REDIS_URL not set — caching and rate limiting disabled")
		cacheStore = &cache.NoOpCache{}
	} else {
		opt, parseErr := redis.ParseURL(redisURL)
		if parseErr != nil {
			log.Printf("doctor-service: WARNING: invalid REDIS_URL %q: %v — caching disabled", redisURL, parseErr)
			cacheStore = &cache.NoOpCache{}
		} else {
			rdb := redis.NewClient(opt)
			pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if pingErr := rdb.Ping(pingCtx).Err(); pingErr != nil {
				log.Printf("doctor-service: WARNING: cannot reach Redis at %s: %v — caching disabled", redisURL, pingErr)
				cacheStore = &cache.NoOpCache{}
			} else {
				log.Printf("doctor-service: connected to Redis at %s", redisURL)
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
		log.Println("doctor-service: NATS_URL not set — events will not be published")
		publisher = &event.NoOpPublisher{}
	} else {
		p, err := event.NewNATSPublisher(natsURL)
		if err != nil {
			log.Printf("doctor-service: WARNING: cannot connect to NATS at %s: %v — continuing without event publishing", natsURL, err)
			publisher = &event.NoOpPublisher{}
		} else {
			publisher = p
			log.Printf("doctor-service: connected to NATS at %s", natsURL)
		}
	}

	// ── Wire up layers ──────────────────────────────────────────────────────
	pgRepo := repository.NewPostgresDoctorRepository(db)
	cachedRepo := cache.NewCachedDoctorRepository(pgRepo, cacheStore, cacheTTL)
	uc := usecase.NewDoctorUseCase(cachedRepo, publisher)
	server := transportgrpc.NewDoctorServer(uc)

	// ── gRPC server options ──────────────────────────────────────────────────
	var serverOpts []grpc.ServerOption
	if redisClient != nil {
		rl := middleware.NewRateLimiter(redisClient)
		serverOpts = append(serverOpts, grpc.UnaryInterceptor(rl.UnaryServerInterceptor()))
		log.Println("doctor-service: rate limiter interceptor enabled")
	} else {
		log.Println("doctor-service: rate limiter disabled (Redis unavailable)")
	}

	// ── gRPC server ─────────────────────────────────────────────────────────
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("doctor-service: failed to listen on port %s: %v", port, err)
	}

	grpcServer := grpc.NewServer(serverOpts...)
	pb.RegisterDoctorServiceServer(grpcServer, server)
	reflection.Register(grpcServer)

	log.Printf("Doctor Service gRPC listening on :%s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("doctor-service: failed to serve: %v", err)
	}
}
