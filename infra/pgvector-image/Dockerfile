# Use the pgvector base image
FROM ankane/pgvector:v0.5.1

# Set environment variables for PostgreSQL
ENV POSTGRES_USER=postgres
ENV POSTGRES_DB=postgres
ENV POSTGRES_HOST_AUTH_METHOD=trust

# Add initialization script
COPY init_pgvector.sh /docker-entrypoint-initdb.d/

# Ensure the script has the correct permissions
RUN chmod +x /docker-entrypoint-initdb.d/init_pgvector.sh