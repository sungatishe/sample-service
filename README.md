# Project Overview

This project is a multi-service application that includes various components such as a frontend, broker, authentication, logger, mail, and listener services. The application is containerized using Docker, and you can easily build and manage the services through the provided `Makefile`.

## Prerequisites

Ensure you have the following installed on your system:
- Docker
- Docker Compose
- Go (for building the Go services)

### Start the Application

Start all Docker containers in the background without forcing a rebuild:

```bash
make up_build
make start
