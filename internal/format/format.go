package format

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Location struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
	Distance  *int     `json:"distance"`
}

type Line struct {
	Name string `json:"name"`
}

type Stopover struct {
	When            string   `json:"when"`
	PlannedWhen     string   `json:"plannedWhen"`
	Delay           *int     `json:"delay"`
	Platform        string   `json:"platform"`
	PlannedPlatform string   `json:"plannedPlatform"`
	Cancelled       bool     `json:"cancelled"`
	Direction       string   `json:"direction"`
	Line            Line     `json:"line"`
	Stop            Location `json:"stop"`
}

type JourneysResponse struct {
	Journeys []Journey `json:"journeys"`
}

type Journey struct {
	Legs      []Leg `json:"legs"`
	Transfers int   `json:"transfers"`
}

type Leg struct {
	Origin      *Location `json:"origin"`
	Destination *Location `json:"destination"`
	Departure   string    `json:"departure"`
	PlannedDep  string    `json:"plannedDeparture"`
	Arrival     string    `json:"arrival"`
	PlannedArr  string    `json:"plannedArrival"`
}

type TripResponse struct {
	Trip Trip `json:"trip"`
}

type Trip struct {
	Line      Line       `json:"line"`
	Stopovers []TripStop `json:"stopovers"`
}

type TripStop struct {
	Stop             Location `json:"stop"`
	Arrival          string   `json:"arrival"`
	PlannedArrival   string   `json:"plannedArrival"`
	Departure        string   `json:"departure"`
	PlannedDeparture string   `json:"plannedDeparture"`
	Platform         string   `json:"platform"`
	PlannedPlatform  string   `json:"plannedPlatform"`
}

type RadarResponse struct {
	Movements []Movement `json:"movements"`
}

type Movement struct {
	Line      Line     `json:"line"`
	Direction string   `json:"direction"`
	Location  Position `json:"location"`
}

type Position struct {
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}

// LocationsPlain formats /locations responses into line-based text.
func LocationsPlain(data []byte, withHeader bool) (string, error) {
	var locations []Location
	if err := json.Unmarshal(data, &locations); err != nil {
		return "", err
	}
	if len(locations) == 0 {
		if withHeader {
			return "no results\n", nil
		}
		return "", nil
	}
	var b strings.Builder
	if withHeader {
		b.WriteString("id\tname\ttype\tlatitude\tlongitude\tdistance_m\n")
	}
	for _, loc := range locations {
		b.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\n",
			loc.ID,
			loc.Name,
			loc.Type,
			formatFloat(loc.Latitude),
			formatFloat(loc.Longitude),
			formatInt(loc.Distance),
		))
	}
	return b.String(), nil
}

// StopoversPlain formats departures/arrivals into line-based text.
func StopoversPlain(data []byte, withHeader bool) (string, error) {
	var stopovers []Stopover
	if err := json.Unmarshal(data, &stopovers); err != nil {
		return "", err
	}
	if len(stopovers) == 0 {
		if withHeader {
			return "no results\n", nil
		}
		return "", nil
	}
	var b strings.Builder
	if withHeader {
		b.WriteString("time\tline\tdirection\tplatform\tdelay\tstatus\n")
	}
	for _, s := range stopovers {
		timeValue := pickTime(s.When, s.PlannedWhen)
		platform := pickString(s.Platform, s.PlannedPlatform)
		status := "-"
		if s.Cancelled {
			status = "cancelled"
		}
		b.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\n",
			timeValue,
			s.Line.Name,
			s.Direction,
			platform,
			formatDelay(s.Delay),
			status,
		))
	}
	return b.String(), nil
}

// JourneysPlain formats /journeys responses into line-based text.
func JourneysPlain(data []byte, withHeader bool) (string, error) {
	var resp JourneysResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", err
	}
	if len(resp.Journeys) == 0 {
		if withHeader {
			return "no results\n", nil
		}
		return "", nil
	}
	var b strings.Builder
	if withHeader {
		b.WriteString("departure\torigin\tarrival\tdestination\ttransfers\n")
	}
	for _, journey := range resp.Journeys {
		if len(journey.Legs) == 0 {
			continue
		}
		first := journey.Legs[0]
		last := journey.Legs[len(journey.Legs)-1]
		origin := locationName(first.Origin)
		destination := locationName(last.Destination)
		departure := pickTime(first.Departure, first.PlannedDep)
		arrival := pickTime(last.Arrival, last.PlannedArr)
		b.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%d\n",
			departure,
			origin,
			arrival,
			destination,
			journey.Transfers,
		))
	}
	return b.String(), nil
}

// TripPlain formats /trips/{id} responses into line-based text.
func TripPlain(data []byte, withHeader bool) (string, error) {
	var resp TripResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", err
	}
	if len(resp.Trip.Stopovers) == 0 {
		if withHeader {
			return "no results\n", nil
		}
		return "", nil
	}
	var b strings.Builder
	if withHeader {
		b.WriteString("line\tstop\tarrival\tdeparture\tplatform\n")
	}
	for _, stop := range resp.Trip.Stopovers {
		arrival := pickTime(stop.Arrival, stop.PlannedArrival)
		departure := pickTime(stop.Departure, stop.PlannedDeparture)
		platform := pickString(stop.Platform, stop.PlannedPlatform)
		b.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%s\n",
			resp.Trip.Line.Name,
			stop.Stop.Name,
			arrival,
			departure,
			platform,
		))
	}
	return b.String(), nil
}

// RadarPlain formats /radar responses into line-based text.
func RadarPlain(data []byte, withHeader bool) (string, error) {
	var resp RadarResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", err
	}
	if len(resp.Movements) == 0 {
		if withHeader {
			return "no results\n", nil
		}
		return "", nil
	}
	var b strings.Builder
	if withHeader {
		b.WriteString("line\tdirection\tlatitude\tlongitude\n")
	}
	for _, movement := range resp.Movements {
		b.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\n",
			movement.Line.Name,
			movement.Direction,
			formatFloat(movement.Location.Latitude),
			formatFloat(movement.Location.Longitude),
		))
	}
	return b.String(), nil
}

func formatFloat(value *float64) string {
	if value == nil {
		return "-"
	}
	return fmt.Sprintf("%.6f", *value)
}

func formatInt(value *int) string {
	if value == nil {
		return "-"
	}
	return fmt.Sprintf("%d", *value)
}

func formatDelay(delay *int) string {
	if delay == nil {
		return "-"
	}
	if *delay == 0 {
		return "0m"
	}
	if *delay%60 == 0 {
		return fmt.Sprintf("%+dm", *delay/60)
	}
	return fmt.Sprintf("%+ds", *delay)
}

func pickString(primary, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}
	if strings.TrimSpace(fallback) != "" {
		return fallback
	}
	return "-"
}

func pickTime(primary, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}
	if strings.TrimSpace(fallback) != "" {
		return fallback
	}
	return "-"
}

func locationName(loc *Location) string {
	if loc == nil {
		return "-"
	}
	if strings.TrimSpace(loc.Name) != "" {
		return loc.Name
	}
	return loc.ID
}
