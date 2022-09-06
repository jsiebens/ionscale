# Build
```shell
docker build -t $USER/ionscale:dev-latest -t $USER/ionscale:dev-$(git rev-parse --short HEAD) . -f Dockerfile
```
This will build a docker image tagged with your USER and Labels with the current commit and "dev-latest"

# Run
```shell
docker network create --attachable net-ionscale
docker run -d --name ionscale --net net-ionscale -v $(pwd)/conf:/data/conf -v $(pwd)/db:/data/db -v $(pwd)/policies:/data/policies $USER/ionscale:dev-latest
```
This will run our newly crafted docker image on an isolated network with 3 volumes: configurations, database, and policies.  
The default entrypoint is `/usr/local/bin/ionscale`  
The default command run is `server --config /data/conf/ionscale.yaml`  
This yields `/usr/local/bin/ionscale server --config /data/conf/ionscale.yaml`
Ports can be exposed by adding `-p <external>:<internal>/<proto>`  
Default configs would be `-p 8080:8080/tcp -p 8443:8443/tcp -p 8081:8081/tcp` but are left as an exercise to the user