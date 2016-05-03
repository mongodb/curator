/*
Package operations contains the integration between the core Curator
functionality, and the user-exposed interfaces.

The public functions in this package return either cli.Command objects
or http.HandlerFunc functions that are integrated into the correct
interfaces. Additionally, there may be a number of private functions
that integrate between components (queues, Job definitions, etc.)

In general core business logic will be attached to the amboy.Job
implementations and the models for configuration objects.
*/
package operations

// This file is documentation only.
