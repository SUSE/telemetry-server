# Use the latest upstream postgres:16 image
FROM postgres:16

# add the telemetry init scripting
ADD telemetry /telemetry
RUN chmod +x /telemetry/*.bash
