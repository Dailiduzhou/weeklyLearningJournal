CREATE TABLE "user" (
  id bigserial PRIMARY KEY,
  username VARCHAR(255) NOT NULL,
  password VARCHAR(255) NOT NULL,
  role VARCHAR(255) NOT NULL
);

CREATE INDEX idx_user ON "user"(username);
