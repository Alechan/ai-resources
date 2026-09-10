package datadogapi

import "testing"

func TestAPIErrorFromResponse_IncludesServerErrors(t *testing.T) {
	t.Parallel()

	// Given
	body := `{"errors":["invalid cell definition","missing graph_size"]}`

	// When
	err := apiErrorFromResponse(400, body)

	// Then
	if err.Message != "invalid cell definition" {
		t.Fatalf("Message = %q", err.Message)
	}
	if err.HTTPStatus != 400 {
		t.Fatalf("HTTPStatus = %d", err.HTTPStatus)
	}
	if len(err.Errors) != 2 {
		t.Fatalf("Errors = %v", err.Errors)
	}
	if err.Details != body {
		t.Fatalf("Details = %q", err.Details)
	}
}

func TestAPIErrorFromResponse_StandardBadRequestShape(t *testing.T) {
	t.Parallel()

	// Given
	body := `{"errors":["Bad Request"]}`

	// When
	err := apiErrorFromResponse(400, body)

	// Then
	if err.Message != "Bad Request" {
		t.Fatalf("Message = %q", err.Message)
	}
	if len(err.Errors) != 1 || err.Errors[0] != "Bad Request" {
		t.Fatalf("Errors = %v", err.Errors)
	}
	if err.Details != body {
		t.Fatalf("Details = %q", err.Details)
	}
}

func TestAPIErrorFromResponse_NonJSONBody(t *testing.T) {
	t.Parallel()

	// Given
	body := "Bad Request"

	// When
	err := apiErrorFromResponse(400, body)

	// Then
	if err.Message != "Bad Request" {
		t.Fatalf("Message = %q", err.Message)
	}
	if len(err.Errors) != 0 {
		t.Fatalf("Errors = %v", err.Errors)
	}
	if err.Details != body {
		t.Fatalf("Details = %q", err.Details)
	}
}

func TestAPIErrorFromResponse_EmptyBody(t *testing.T) {
	t.Parallel()

	// When
	err := apiErrorFromResponse(400, "")

	// Then
	if err.Message != "HTTP 400" {
		t.Fatalf("Message = %q", err.Message)
	}
	if len(err.Errors) != 0 {
		t.Fatalf("Errors = %v", err.Errors)
	}
	if err.Details != "" {
		t.Fatalf("Details = %q", err.Details)
	}
}

func TestAPIErrorFromResponse_TopLevelErrorField(t *testing.T) {
	t.Parallel()

	// Given
	body := `{"error":"notebook payload invalid"}`

	// When
	err := apiErrorFromResponse(422, body)

	// Then
	if err.Message != "notebook payload invalid" {
		t.Fatalf("Message = %q", err.Message)
	}
	if len(err.Errors) != 1 || err.Errors[0] != "notebook payload invalid" {
		t.Fatalf("Errors = %v", err.Errors)
	}
}
