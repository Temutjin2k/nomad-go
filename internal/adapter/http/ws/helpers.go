package wshandler

import (
	ws "github.com/Temutjin2k/ride-hail-system/pkg/wsHub"
)

func errorResponse(conn *ws.Conn, message any) error {
	return conn.Send(
		map[string]any{
			"error": message,
		})
}

func failedValidationResponse(conn *ws.Conn, errors map[string]string) error {
	return errorResponse(conn, errors)
}
