# dbHandler
Manages the interface to firestore db <br>

All data passed in and out must be primitive types or arrays. <br>
All APIs need to return a result and an error string <br>

## Testing
Tests require the firestore emulators to be running<br>
**firebase emulators:start --project drjim-f2087** <br>

It will start the emulator automatically if not running, but firebase has to be already installed <br>
<br>
To view the firestore content **http://127.0.0.1:4000/firestore/**<br>

