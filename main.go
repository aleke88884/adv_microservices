package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

func main() {
	fmt.Println("Starting Medical Scheduling Platform (gRPC)...")
	fmt.Println("Doctor Service gRPC:      localhost:50051")
	fmt.Println("Appointment Service gRPC: localhost:50052")
	fmt.Println("\nPress Ctrl+C to stop all services\n")

	doctorCmd := exec.Command("go", "run", "./doctor-service/cmd/doctor-service/main.go")
	doctorCmd.Stdout = os.Stdout
	doctorCmd.Stderr = os.Stderr

	appointmentCmd := exec.Command("go", "run", "./appointment-service/cmd/appointment-service/main.go")
	appointmentCmd.Stdout = os.Stdout
	appointmentCmd.Stderr = os.Stderr

	if err := doctorCmd.Start(); err != nil {
		fmt.Printf("Failed to start Doctor Service: %v\n", err)
		return
	}

	if err := appointmentCmd.Start(); err != nil {
		fmt.Printf("Failed to start Appointment Service: %v\n", err)

		_ = doctorCmd.Process.Signal(syscall.SIGTERM)

		return
	}

	waitErrCh := make(chan error, 2)

	go func() {
		if err := doctorCmd.Wait(); err != nil {
			waitErrCh <- fmt.Errorf("doctor service exited with error: %w", err)
			return
		}
		waitErrCh <- fmt.Errorf("doctor service stopped")
	}()

	go func() {
		if err := appointmentCmd.Wait(); err != nil {
			waitErrCh <- fmt.Errorf("appointment service exited with error: %w", err)
			return
		}
		waitErrCh <- fmt.Errorf("appointment service stopped")
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	select {
	case sig := <-sigChan:
		fmt.Printf("\nReceived signal: %v\n", sig)
		fmt.Println("Shutting down services...")
		_ = doctorCmd.Process.Signal(syscall.SIGTERM)
		_ = appointmentCmd.Process.Signal(syscall.SIGTERM)

		// optional: wait for both goroutines to report process exit
		for i := 0; i < 2; i++ {
			fmt.Println(<-waitErrCh)
		}

	case err := <-waitErrCh:
		fmt.Printf("\nProcess finished: %v\n", err)
	}

	fmt.Println("Services stopped")
}
