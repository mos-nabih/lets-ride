CREATE TABLE rides (
    id INTEGER NOT NULL PRIMARY KEY AUTO_INCREMENT,
    title VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,
    created DATETIME NOT NULL,
    departure DATETIME NOT NULL
);

CREATE INDEX idx_rides_created ON rides(created);

CREATE TABLE users (
    id INTEGER NOT NULL PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    hashed_password CHAR(60) NOT NULL,
    created DATETIME NOT NULL,
    CONSTRAINT users_uc_email UNIQUE (email)
);

CREATE TABLE sessions (
    token CHAR(43) NOT NULL PRIMARY KEY,
    data BLOB NOT NULL,
    expiry TIMESTAMP(6) NOT NULL
);

CREATE INDEX idx_sessions_expiry ON sessions(expiry);
