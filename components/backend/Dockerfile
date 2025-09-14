# Simple container with pre-built binary
FROM --platform=linux/amd64 alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /app

# Copy the pre-built binary
COPY main .

# Set executable permissions and make accessible to any user
RUN chmod +x ./main && chmod 755 /app

# Expose port
EXPOSE 8080

# Command to run the executable
CMD ["./main"]
