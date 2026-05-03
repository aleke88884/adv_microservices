package app

import (
	"database/sql"
	"doctor-service/internal/event"
	"doctor-service/internal/repository"
	transportgrpc "doctor-service/internal/transport/grpc"
	"doctor-service/internal/usecase"
	pb "doctor-service/proto"
	"log"
	"net"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
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
	repo := repository.NewPostgresDoctorRepository(db)
	uc := usecase.NewDoctorUseCase(repo, publisher)
	server := transportgrpc.NewDoctorServer(uc)

	// ── gRPC server ─────────────────────────────────────────────────────────
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("doctor-service: failed to listen on port %s: %v", port, err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterDoctorServiceServer(grpcServer, server)
	reflection.Register(grpcServer)

	log.Printf("Doctor Service gRPC listening on :%s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("doctor-service: failed to serve: %v", err)
	}
}
