CREATE TABLE IF NOT EXISTS rooms (
  "id"        uuid          PRIMARY KEY NOT NULL DEFAULT gen_random_uuid(),
  "theme"     varchar(255)              NOT NULL
);

---- create above / drop below ----

DROP TABLE IF EXISTS rooms;
