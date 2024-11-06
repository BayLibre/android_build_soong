#!/bin/bash

#aprotoc --go_out=paths=source_relative:. -I ../../../../../external/protobuf/src:.  internal_state.proto
aprotoc --go_out=paths=source_relative:.  internal_state.proto
