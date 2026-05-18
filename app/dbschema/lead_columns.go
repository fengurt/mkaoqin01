package dbschema

import "database/sql"

// LeadWideSelectColumns is the SELECT list for client_leads (SQLite IFNULL pattern).
// Keep in sync with client_leads columns in leads.sql / postgresql.sql.
const LeadWideSelectColumns = `id, source, IFNULL(source_ref,''), IFNULL(lead_segment,'NEW_PURE'), IFNULL(approx_origin_region,'UNKNOWN'),
       IFNULL(preferred_venue,'UNSPECIFIED'), IFNULL(language_pref,'zh-CN'), estimated_party_size,
       IFNULL(event_occasion,'OTHER'), IFNULL(channel_touchpoint,'UNKNOWN'),
       client_name, IFNULL(client_phone,''), IFNULL(client_company,''), intent, IFNULL(notes,''), priority, status,
       IFNULL(assigned_user_id,0), IFNULL(picked_up_by,0), IFNULL(picked_up_at,''), created_at, updated_at, IFNULL(metadata,''),
       IFNULL(ref_last_visit_at,''), IFNULL(ref_last_property,''), IFNULL(ref_ltv_tier,''), IFNULL(ref_host_name,''),
       IFNULL(ref_member_id_masked,''), IFNULL(ref_notes,'')`

// ClientLeadWideScanner matches *sql.Rows and *sql.Row for Scan.
type ClientLeadWideScanner interface {
	Scan(dest ...any) error
}

// ScanClientLeadWideRow scans one client_leads row (see LeadWideSelectColumns order) into a JSON-friendly map.
func ScanClientLeadWideRow(scanner ClientLeadWideScanner) (map[string]any, error) {
	var id, assigned, pickedUpBy int64
	var source, sourceRef, seg, origin, venue, lang, occasion, channel string
	var est sql.NullInt64
	var name, phone, company, intent, notes, priority, status string
	var pickedAt, createdAt, updatedAt, meta string
	var refVisit, refProp, refTier, refHost, refMask, refNotes string
	if err := scanner.Scan(
		&id, &source, &sourceRef, &seg, &origin, &venue, &lang, &est, &occasion, &channel,
		&name, &phone, &company, &intent, &notes, &priority, &status, &assigned, &pickedUpBy, &pickedAt, &createdAt, &updatedAt, &meta,
		&refVisit, &refProp, &refTier, &refHost, &refMask, &refNotes,
	); err != nil {
		return nil, err
	}
	out := map[string]any{
		"id": id, "source": source, "sourceRef": sourceRef, "leadSegment": seg, "approxOriginRegion": origin,
		"preferredVenue": venue, "languagePref": lang, "eventOccasion": occasion, "channelTouchpoint": channel,
		"clientName": name, "clientPhone": phone, "clientCompany": company, "intent": intent, "notes": notes,
		"priority": priority, "status": status, "assignedUserId": assigned, "pickedUpBy": pickedUpBy, "pickedUpAt": pickedAt,
		"createdAt": createdAt, "updatedAt": updatedAt, "metadata": meta,
		"refLastVisitAt": refVisit, "refLastProperty": refProp, "refLtvTier": refTier, "refHostName": refHost,
		"refMemberIdMasked": refMask, "refNotes": refNotes,
	}
	if est.Valid {
		out["estimatedPartySize"] = est.Int64
	} else {
		out["estimatedPartySize"] = nil
	}
	return out, nil
}
