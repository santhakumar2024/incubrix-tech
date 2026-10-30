FROM node:22-alpine AS frontend
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY index.html tsconfig.json vite.config.ts ./
COPY src ./src
RUN npm run build

FROM golang:1.23-alpine AS backend
WORKDIR /app/server
COPY server/go.mod server/go.sum* ./
RUN go mod download
COPY server/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /seatflow .

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app/server
COPY --from=backend /seatflow /app/seatflow
COPY --from=backend /app/server/schema.sql /app/server/schema.sql
COPY --from=frontend /app/dist /app/dist
ENV WEB_DIST=/app/dist
EXPOSE 8080
ENTRYPOINT ["/app/seatflow"]
