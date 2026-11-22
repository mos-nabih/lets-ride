# Let’s Ride

Let’s Ride gives local motorcyclists one place to find upcoming group rides and share invitations. Anyone can browse the ten most recently posted upcoming rides or open a ride’s link. Signed-in riders can publish an invitation with a title, description, and departure time.

## Run locally

You’ll need Go 1.27 or later, a running MySQL server, the `mysql` client, and OpenSSL. Run the commands from the repository root.

For a fresh database, run the following as a MySQL administrator. Replace `local-password` with your own local password.

```sql
CREATE DATABASE lets_ride;
CREATE USER 'lets_ride'@'127.0.0.1' IDENTIFIED BY 'local-password';
GRANT SELECT, INSERT, UPDATE, DELETE ON lets_ride.* TO 'lets_ride'@'127.0.0.1';
```

Apply the schema once using your MySQL administrator account.

```sh
mysql -u root -p lets_ride < schema.sql
```

If you need a local HTTPS certificate, generate one for `localhost`.

```sh
mkdir -p assets/tls
openssl req -x509 -newkey rsa:2048 -nodes -days 30 \
  -keyout assets/tls/key.pem -out assets/tls/cert.pem \
  -subj '/CN=localhost' -addext 'subjectAltName=DNS:localhost'
```

Start the server with the database account you created. Keep `parseTime=true` so the driver reads MySQL date and time values as Go timestamps.

```sh
export DATABASE_DSN='lets_ride:local-password@tcp(127.0.0.1:3306)/lets_ride?parseTime=true'
go run ./cmd/web
```

Open [https://localhost:4000](https://localhost:4000). A browser may ask you to trust the self-signed certificate. Use `-addr` to choose a different listen address or `-dsn` to override `DATABASE_DSN`.

## Sample data

Load [seed.sql](seed.sql) into your local development database after applying the schema.

```sh
mysql -h 127.0.0.1 -u lets_ride -p lets_ride < seed.sql
```

The seed adds three rides departing in 1, 3, and 7 days, plus one that already departed. The upcoming rides appear on the home page. The departed ride is hidden and its detail URL returns 404. Rerunning the seed skips existing sample entries and preserves their original dates.

Sign in with `demo@example.com` and the demo password `ride-demo-2026` to try posting a ride.

## Go architecture

A single Go server serves embedded CSS and renders pages with `html/template`. Dynamic routes load sessions, check CSRF tokens, and identify the signed-in user. Posting a ride also requires authentication.

```text
Browser
  |
  | HTTPS
  v
Go net/http.ServeMux
  |
  +-- /static/ --> Embedded CSS
  |
  +-- Dynamic routes --> Session, CSRF, and authentication middleware
                           |
                           +-- Session store --> MySQL sessions
                           |
                           +-- Handlers
                                 +-- Embedded HTML templates
                                 +-- Models --> database/sql --> MySQL rides and users
```

## Ride lifecycle

The server interprets and displays departures in its local time zone. Set `TZ` when starting the server to choose the board’s time zone. MySQL stores ride times in UTC and checks departure times on each public read.

```text
Signed-in rider submits an invitation
  |
  v
Server validates the form and saves the departure in UTC
  |
  +-- Before departure --> Eligible for the home page and visible by link
  |
  +-- At or after departure --> Hidden from the home page, detail URL returns 404
```

## Tests

```sh
go test ./...
```

To include the MySQL integration tests, set `TEST_DATABASE_DSN` to a dedicated, empty test database and include `parseTime=true&multiStatements=true` in the DSN. The test account needs permission to create and drop tables because the tests apply and tear down the schema.
