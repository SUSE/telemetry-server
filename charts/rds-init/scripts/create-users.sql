    -- Create the telemetry user
    SELECT 'CREATE USER telemetry WITH PASSWORD ''__TELEMETRY_PASSWORD__'''
    WHERE NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'telemetry')\gexec

    -- Create the bi_team user
    SELECT 'CREATE USER bi_team WITH PASSWORD ''__BI_TEAM_PASSWORD__'''
    WHERE NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'bi_team')\gexec

    -- Create operational database and grant user access
    SELECT 'CREATE DATABASE operational WITH TEMPLATE template0'
    WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname= 'operational')\gexec
    \c operational
    GRANT ALL ON SCHEMA public TO telemetry;
    GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO telemetry;
    ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO telemetry;

    -- Create telemetry database and grant user access
    SELECT 'CREATE DATABASE telemetry WITH TEMPLATE template0'
    WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname= 'telemetry')\gexec
    \c telemetry
    GRANT ALL ON SCHEMA public TO telemetry;
    GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO telemetry;
    ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO telemetry;
    GRANT ALL PRIVILEGES ON TABLE telemetrydata TO "bi_team";