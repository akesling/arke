// Package arke is an ultralight pub-sub system designed to be used as a message
// routing core for loosely coupled applications.
//
// See package "arke/interchange" for the core API and "arke/server" for details
// related to providing extra-process interfaces, including a daemonizeable Arke
// server.
//
// The Hub
//
// The central element of Arke is the hub.  The hub manages all system events,
// whether that be publication, subscription, cancellation, what-have-you.  The
// primary means of interacting on a hub is via a Client interface.
//
// Topics
//
// Topics in Arke are dot-delimeted strings like
//	"foo.bar.baz"
//
// These strings form a hierachy used during publication.  For example, if there
// exist two subscribers, Subscriber A on "foo.bar" and Subscriber B on
// "foo.baz".  When a Publisher C publishes on "foo" then both A and B will
// receive this publication.
//
// Performance
//
// Arke is optimized for publication.  This means that when there is a
// performance trade-off between supporting publication or subscription,
// publication will win out.
package arke
