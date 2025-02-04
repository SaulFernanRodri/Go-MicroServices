# Microservices Project

This project is a microservices-based application built with Go. It consists of several services, each responsible for a specific functionality. The services communicate with each other using HTTP and gRPC.

## Project Structure

## Services

### Authentication Service

Handles user authentication and authorization.

### Broker Service

Acts as a message broker for inter-service communication.

### Front-End

The front-end service that provides the user interface.

### Listener Service

Listens for events and processes them accordingly.

### Logger Service

Handles logging for the entire application.

### Mail Service

Handles sending emails.

## Getting Started

### Prerequisites

- Go 1.22.2 or later
- Docker

### Building the Project

To build the project, run the following command:

```sh
make up_build
```

### Running the Project

To start the project, run the following command:

```sh
make up
```

### Stopping the Project

To stop the project, run the following command:


```sh
make down
```