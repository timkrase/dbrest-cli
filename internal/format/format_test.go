package format

import "testing"

func TestLocationsPlain(t *testing.T) {
	data := []byte(`[{"id":"123","name":"Berlin Hbf","type":"station","latitude":52.525,"longitude":13.369,"distance":120}]`)
	out, err := LocationsPlain(data, true)
	if err != nil {
		t.Fatalf("LocationsPlain error: %v", err)
	}
	expected := "id\tname\ttype\tlatitude\tlongitude\tdistance_m\n" +
		"123\tBerlin Hbf\tstation\t52.525000\t13.369000\t120\n"
	if out != expected {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestStopoversPlain(t *testing.T) {
	data := []byte(`[{"when":"2024-01-01T12:00:00+01:00","line":{"name":"S1"},"direction":"Frohnau","platform":"1","delay":120,"cancelled":false}]`)
	out, err := StopoversPlain(data, true)
	if err != nil {
		t.Fatalf("StopoversPlain error: %v", err)
	}
	expected := "time\tline\tdirection\tplatform\tdelay\tstatus\n" +
		"2024-01-01T12:00:00+01:00\tS1\tFrohnau\t1\t+2m\t-\n"
	if out != expected {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestStopoversPlainEnvelope(t *testing.T) {
	data := []byte(`{"departures":[{"when":"2024-01-01T12:00:00+01:00","line":{"name":"S1"},"direction":"Frohnau","platform":"1","delay":0,"cancelled":false}]}`)
	out, err := StopoversPlain(data, true)
	if err != nil {
		t.Fatalf("StopoversPlain error: %v", err)
	}
	expected := "time\tline\tdirection\tplatform\tdelay\tstatus\n" +
		"2024-01-01T12:00:00+01:00\tS1\tFrohnau\t1\t0m\t-\n"
	if out != expected {
		t.Fatalf("unexpected output:\n%s", out)
	}
}
