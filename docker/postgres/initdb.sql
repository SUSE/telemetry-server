-- Create the telemetry user
CREATE USER telemetry;

-- Create staging database and grant telemetry user access
CREATE DATABASE staging WITH TEMPLATE template0;
GRANT ALL PRIVILEGES ON DATABASE staging TO telemetry;

-- Create operational database and grant telemetry user access
CREATE DATABASE operational WITH TEMPLATE template0;
GRANT ALL PRIVILEGES ON DATABASE operational TO telemetry;

-- Create telemetry database and grant telemetry user access
CREATE DATABASE telemetry WITH TEMPLATE template0;
GRANT ALL PRIVILEGES ON DATABASE telemetry TO telemetry;
