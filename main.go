package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

func main() {
	fmt.Println("Starting Medical Scheduling Platform...")
	fmt.Println("Doctor Service: http://localhost:8080")
	fmt.Println("Appointment Service: http://localhost:8081")
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

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan

	fmt.Println("\nShutting down services...")
	_ = doctorCmd.Process.Signal(syscall.SIGTERM)
	_ = appointmentCmd.Process.Signal(syscall.SIGTERM)
	fmt.Println("Services stopped")
}
