FROM golang:1.22

# Install cron
RUN apt-get update && apt-get install -y cron

# Copy your application files to the container
COPY . /app
WORKDIR /app

# Build your application
RUN make build

# Add a cron job to run the application every 5 minutes
RUN (crontab -l ; echo "*/2 * * * * cd /app && ./build/supernova -sub-accounts 5 -transactions 100 -url http://gno_node:26657 -mnemonic 'source bonus chronic canvas draft south burst lottery vacant surface solve popular case indicate oppose farm nothing bullet exhibit title speed wink action roast' -output result.json >> /var/log/cron.log 2>&1") | crontab

# Run the cron daemon when the container starts
CMD ["bash", "-c", "cd /app && ./build/supernova -sub-accounts 5 -transactions 100 -url http://gno_node:26657 -mnemonic 'source bonus chronic canvas draft south burst lottery vacant surface solve popular case indicate oppose farm nothing bullet exhibit title speed wink action roast' -output result.json && cron -f"]
