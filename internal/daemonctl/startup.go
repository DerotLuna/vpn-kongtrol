package daemonctl

import (
	"net/http"
)

// AnswerRestoreQuestion answers "¿Volver a conectar tus VPN al encender el
// equipo?" (PUT /api/v1/startup {"restore_answer": on}, ADR-0101): the
// daemon saves it and every open panel and tray flyout drops the question.
func (c *Client) AnswerRestoreQuestion(on bool) error {
	resp, err := c.sendJSON(http.MethodPut, "/api/v1/startup", map[string]bool{"restore_answer": on})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return APIError(resp)
	}
	return nil
}
