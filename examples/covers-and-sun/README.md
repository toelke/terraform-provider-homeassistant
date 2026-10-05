# Covers and the sun, written in HCL

These automations are written directly in HCL instead of YAML, using the provider's builder
functions `numeric_state_trigger`, `state_trigger`, `time_condition` and `action`. Each function
returns the same plain object you would otherwise write by hand, so they mix freely with literal
objects for trigger and condition types that have no builder (`sun`, `numeric_state` and `or`
conditions here), and shared values such as the weekday lists or the sun azimuth limit live in
`locals`. The office blind goes down while the morning sun shines on the screens and comes up
again once it has moved on; the terrace awning retracts an hour before sunset; the first light on
the ground floor in the morning opens the patio blind, with the lights of each ground-floor area
looked up through the `homeassistant_entities` data source; and the receiver going off late in
the evening lights the hallway.
