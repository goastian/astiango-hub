# Build stage
FROM node:20-alpine AS build

# Arguments and environment variables for building packages
ARG VITE_APP_API_BASE_URL=/api
ENV VITE_APP_API_BASE_URL ${VITE_APP_API_BASE_URL}

# Set the working directory in the container
WORKDIR /app

# Install pnpm
RUN npm install -g pnpm

# Copy application code
ADD ./astiango-hub-ui /app

# Debug: List files to verify they're correctly copied
RUN ls -la /app/src/
RUN ls -la /app/src/components/
RUN ls -la /app/src/components/ui/ || echo "UI directory not found"

# Install project dependencies
RUN pnpm install

# Build the application
RUN pnpm run build

# Production stage
FROM alpine:3.14

# Copy only the built artifacts from the build stage
COPY --from=build /app/dist /app/dist
