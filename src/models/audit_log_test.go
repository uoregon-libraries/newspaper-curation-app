package models

import "testing"

func TestAuditActionToString(t *testing.T) {
	for action := AuditActionUnderflow + 1; action < AuditActionOverflow; action++ {
		var expected = dbAuditActions[action]
		var got = action.String()
		if expected != got {
			t.Errorf("Expected %q, got %#v", expected, got)
		}
	}
}

func TestBuildAuditLog(t *testing.T) {
	var alog, err = BuildAuditLog("ip", "user", AuditActionClaim, "message")
	if err != nil {
		t.Fatalf("Got error building log: %s", err)
	}
	if alog.IP != "ip" {
		t.Fatalf("Got IP %q, expected %q", alog.IP, "ip")
	}
	if alog.User != "user" {
		t.Fatalf("Got user %q, expected %q", alog.User, "user")
	}
	if alog.Action != "claim" {
		t.Fatalf("Got action %q, expected %q", alog.Action, "claim")
	}
	if alog.Message != "message" {
		t.Fatalf("Got message %q, expected %q", alog.Message, "ip")
	}
}
