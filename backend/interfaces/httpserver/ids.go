package httpserver

import (
	"net/http"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// parseObjectIDField parses a hex ObjectID that came from a JSON
// request body field (as opposed to a route {param} — see
// pathObjectID) — e.g. createCycleRequest.FarmID.
func parseObjectIDField(hex string) (bson.ObjectID, error) {
	return bson.ObjectIDFromHex(hex)
}

// pathObjectID parses the named {param} from the request's route
// pattern as a bson.ObjectID, returning ok=false (and having already
// written a 400 response) if it's missing or malformed — handlers can
// just `return` on ok==false without writing their own error.
func pathObjectID(w http.ResponseWriter, r *http.Request, name string) (bson.ObjectID, bool) {
	id, err := bson.ObjectIDFromHex(r.PathValue(name))
	if err != nil {
		writeError(w, errMissingPathParam)
		return bson.ObjectID{}, false
	}
	return id, true
}
