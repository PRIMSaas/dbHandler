# dbHandler
Manages the interface to firestore db <br>

All data passed in and out must be primitive types or arrays. <br>
All APIs need to return a result and an error string <br>

## Testing
Tests require the firestore emulators to be running<br>
**firebase emulators:start --project drjim-f2087** <br>

It will start the emulator automatically if not running, but firebase has to be already installed <br>

## Local container testing

Build code with **make** <br>
build container with **docker build -t drjimdb .** <br>
Run container locally **docker compose up** <br><br>
Install into cloud run: **make deploy**

## Call APIs for testing
Suggester order: 
* Run code locally in debugger
* containerise and run in local docker
* deploy to cloud run
<br><br>
Test APIS - beware in cloud run you leave out the port number: <br>

curl -i -d '{"fileContent":"This is an awesome file ...", "serviceCodes":["one", "two"]}' -X POST http://localhost:8088/processFile <br>

curl -i -d '{"userId":"srfsyfuqPXfhigSTTkJFwvBs9Jb2"}' -X POST https://drjimdb3-5f6uwrh2eq-km.a.run.app/getCompanies <br>
curl -i -d '{"userId":"srfsyfuqPXfhigSTTkJFwvBs9Jb2"}' -X POST https://drjimdb-5f6uwrh2eq-ts.a.run.app/getCompanies<br>

<br>
To view the firestore content **http://127.0.0.1:4000/firestore/**<br>

## Dockering
do a **make clean** first! <br>
docker build -t drjimdb . <br>
verbose: docker build --no-cache --progress=plain -t drjimdb . <br>
clean up the mess: docker build --rm or docker rmi $(docker images -f “dangling=true” -q) <br>
gcp<br>
docker tag drjimdb australia-southeast2-docker.pkg.dev/drjim-f2087/drjimrepo/drjimdb <br>
docker push australia-southeast2-docker.pkg.dev/drjim-f2087/drjimrepo/drjimdb <br>
gcloud artifacts docker images list  australia-southeast2-docker.pkg.dev/drjim-f2087/drjimrepo/drjimdb <br>
running<br>
command line: <br>
./dbHandler ./config <br>
container run: <br>
docker compose up <br>
docker run --entrypoint /bin/bash -it drjimdb <br>


