# Webhooks Developer Documentation

## Overview

Theres three main systems/abstractions in place regarding webhooks as explained below:

## core

This contains core structs and infrastructure related to webhooks. While most of core is pretty self-explanatory to anyone with basic Go knowledge, there are a few key points to note:

Within ``core``, there are a couple of main/key structures. The first is the ``Event Registry``  (see ``core/events/events.go``). The event registry
keeps track of all registered events and also keeps track of other data produced during event registration such as the ``Test Webhook`` variables created by the deconstructing of each event.

The consequences of the Event Registry and event registration is that **all types used within events must be handled in the deconstring of events (see ``registerEventImpl`` ) as well as on the website's Test Webhook component**

Next, the ``WebhookEvent`` interface (``core/core.go``) serves as a base abstraction for events. Do not make changes to this interface lightly as all events will also need to be changed.

Lastly, we have the ``Driver`` interface (``core/core.go``). As there may be many different types of events, each entity type will need to implement a driver under the ``hooks`` folder. The driver is responsible for constructing the entity target, sending metadata (such as if resuming sends are possible) and may have expanded functionality in the future.