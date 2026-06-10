package check

// ProgressCallback is called with progress updates during proxy checking.
// Set from platform-specific code (e.g. Windows shows a native progress bar).
// phase: "alive", "media", "speed"
// percent: 0-100 overall pipeline progress
// status: human-readable one-line summary
var ProgressCallback func(phase string, percent int, status string)