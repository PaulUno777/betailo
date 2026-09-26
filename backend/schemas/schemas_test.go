package schemas

import "testing"

// TestLoad_AllNamesParse catches the cheap-but-real failure mode for
// this package: a typo in a collection name, or a JSON syntax error in
// one of the embedded files, that would otherwise only surface at
// mongodb.Bootstrap time against a live database.
func TestLoad_AllNamesParse(t *testing.T) {
	if len(Names) == 0 {
		t.Fatal("Names is empty")
	}
	for _, name := range Names {
		t.Run(name, func(t *testing.T) {
			schema, err := Load(name)
			if err != nil {
				t.Fatalf("Load(%q) error = %v", name, err)
			}
			if schema["bsonType"] != "object" {
				t.Errorf("Load(%q) bsonType = %v, want \"object\"", name, schema["bsonType"])
			}
			if _, ok := schema["properties"]; !ok {
				t.Errorf("Load(%q) has no \"properties\" key", name)
			}
		})
	}
}

func TestLoad_UnknownName(t *testing.T) {
	if _, err := Load("does_not_exist"); err == nil {
		t.Error("Load(\"does_not_exist\") error = nil, want an error")
	}
}
