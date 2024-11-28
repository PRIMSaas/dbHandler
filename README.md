# dbHandler
Performs all the calculations required to produce invoice related documents.<br> 

## Local container testing

Build code with **make** <br>
build container with **docker build -t drjimdb .** <br>
Run container locally **docker compose up** <br><br>
Install into cloud run: **make cloud**

## Call APIs for testing
Suggester order: 
* Run code locally in debugger
* containerise and run in local docker
* deploy to cloud run
<br><br>
Test APIS - beware in cloud run you leave out the port number: <br>

curl -i -d '{"fileContent":"Vermont Medical Clinic,Dr Fiona Chao,Irrelevant,Patient Name,164055,176395,72939,36,\"Surgery consultation, Level C\",Payment,26/02/2024,Direct Credit,Medicare,8,80.1,0", "serviceCodes":["one", "two"]}' -X POST http://3.27.224.4:8088/processFile <br>

curl -i -d @curly.json -X POST https://drjimdb-5f6uwrh2eq-ts.a.run.app/processFile<br>

### Testing CORS (OPTIONS)
To let the front end call the API, CORS must be enabled, which sends a OPTIONS http request. To test manually use: <br>
curl -i -H "Origin: http://127.0.0.1:5055" -H "Access-Control-Request-Method: POST" -H "Access-Control-Request-Headers: content-type" -X OPTIONS http://localhost:8088/processFile
 <br>
curl -i -H "Origin: http://127.0.0.1:5055" -H "Access-Control-Request-Method: POST" -H "Access-Control-Request-Headers: content-type" -X OPTIONS https://drjimdb-5f6uwrh2eq-ts.a.run.app/ processFile <br>

<br>
To view the firestore content **http://127.0.0.1:4000/firestore/**<br>

## Dockering
verbose: docker build --no-cache --progress=plain -t drjimdb . <br>
clean up the mess: docker build --rm or docker rmi $(docker images -f “dangling=true” -q) <br>

## gc Clouding
First: Install **gcloud** then run **gcloud init** <br>
Test: gcloud artifacts docker images list  australia-southeast2-docker.pkg.dev/drjim-f2087/drjimrepo/drjimdb <br>

For docker to access the gcp repo run: **gcloud auth print-access-token | docker login -u oauth2accesstoken --password-stdin https://australia-southeast2-docker.pkg.dev**

Now you can do cool things like: <br>
docker tag drjimdb australia-southeast2-docker.pkg.dev/drjim-f2087/drjimrepo/drjimdb <br>
docker push australia-southeast2-docker.pkg.dev/drjim-f2087/drjimrepo/drjimdb <br>
OR <br>
Just run **make cloud** and it's all done for you <br>

## AWS clouding
aws ecr get-login-password --region ap-southeast-2 | docker login --username AWS --password-stdin 600073216458.dkr.ecr.ap-southeast-2.amazonaws.com <br>
docker tag drjimdb:latest 600073216458.dkr.ecr.ap-southeast-2.amazonaws.com/jimrepo:latest <br>
jim@superNUC:~/dev/cloud/dbHandler$ docker push 600073216458.dkr.ecr.ap-southeast-2.amazonaws.com/jimrepo:latest <br>

## Running
### command line:
./dbHandler ./config <br>
### container run:
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

