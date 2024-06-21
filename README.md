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

curl -i -d @curly.json -X POST https://drjimdb-5f6uwrh2eq-ts.a.run.app/processFile<br>

### Testing CORS (OPTIONS)
To let the front end call the API, CORS must be enabled, which sends a OPTIONS http request. To test manually use: <br>
curl -i -H "Origin: http://127.0.0.1:5055" -H "Access-Control-Request-Method: POST" -H "Access-Control-Request-Headers: content-type" -X OPTIONS http://localhost:8088/processFile
 <br>
curl -i -H "Origin: http://127.0.0.1:5055" -H "Access-Control-Request-Method: POST" -H "Access-Control-Request-Headers: content-type" -X OPTIONS https://drjimdb-5f6uwrh2eq-ts.a.run.app/ processFile <br>

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
docker-compose up -d<br>
docker run --entrypoint /bin/bash -it drjimdb <br>

## Calculations to be performed
The input file is layed out like this:<br>
Location 0 A, Provider 1 B, Billed To 2 C, Patient Name 3 D, Invoice No. 4 E, Service ID 5 F, Payment ID 6 G, Item No. 7 H, Description 8 I, Status 9 J, Transaction Date 10 k, Payment Method 11 L, Account Type 12 M, "GST( incl GST)" 13 N, "Payment( incl GST)" 14 O, "Deposit($ incl GST)" 15 P<br>
Need to pass in: <br>
the file content as a string<br>
the line number where the data starts<br>
the name of the company <br>
A CodeMap which is a map of service code to a list of item numbers it covers<br>
A PracMap which is a map of providers to a map of service codes and their respective percentage<br>

The item number in the file is mapped to a service code and the percentage for that service code is given per provider<br>

