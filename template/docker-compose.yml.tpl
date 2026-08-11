services:
  {{.ProjectName}}:
    build: .
    ports:
      - "8080:8080"
    environment:
      APP_ENV: dev
      HTTP_ADDR: 0.0.0.0:8080
      DB_HOST: db
      DB_PORT: "3306"
      DB_NAME: dbv
      DB_USERNAME: db_user
      DB_PASSWORD: db_password!
    depends_on:
      db:
        condition: service_healthy
    restart: unless-stopped

  db:
    image: mysql:8.4
    environment:
      MYSQL_DATABASE: dbv
      MYSQL_USER: db_user
      MYSQL_PASSWORD: db_password!
      MYSQL_ROOT_PASSWORD: root
    ports:
      - "3306:3306"
    volumes:
      - db-data:/var/lib/mysql
    healthcheck:
      # Without this the service starts before MySQL accepts connections.
      test: ["CMD", "mysqladmin", "ping", "-h", "127.0.0.1"]
      interval: 5s
      timeout: 5s
      retries: 20

volumes:
  db-data:
