package buildinfo

// Version is set at build time via -ldflags. Local development builds default
// to "dev".
var Version = "dev"
