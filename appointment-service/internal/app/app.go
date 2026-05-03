package app

import (
	"appointment-service/internal/client"
	"appointment-service/internal/event"
	"appointment-service/internal/repository"
	transportgrpc "appointment-service/internal/transport/grpc"
	"appointment-service/internal/usecase"
	pb "appointment-service/proto"
	"database/sql"
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
	repo := repository.NewPostgresAppointmentRepository(db)
	uc := usecase.NewAppointmentUseCase(repo, doctorClient, publisher)
	server := transportgrpc.NewAppointmentServer(uc)

	// ── gRPC server ─────────────────────────────────────────────────────────
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("appointment-service: failed to listen on port %s: %v", port, err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterAppointmentServiceServer(grpcServer, server)
	reflection.Register(grpcServer)

	log.Printf("Appointment Service gRPC listening on :%s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("appointment-service: failed to serve: %v", err)
	}
}
