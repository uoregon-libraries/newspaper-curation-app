// Package settings just holds global variables all internal packages need to
// access, such as whether debug is on
package settings

// DEBUG is only enabled via command-line and should be used very sparingly,
// such as for user-switching (though an actual user-switch permission would be
// way better)
var DEBUG bool
