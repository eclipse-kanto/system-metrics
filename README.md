[![Kanto logo](https://github.com/eclipse-kanto/kanto/raw/main/logo/kanto.svg)](https://eclipse.dev/kanto/)

# Eclipse Kanto - System Metrics

[![Coverage](https://github.com/eclipse-kanto/system-metrics/wiki/coverage.svg)](#)

This functionality is provided by the Eclipse Kanto as a System Metrics native application.
It allows you to receive the metrics such a CPU utilization, memory usage, and other information for
 supporting system monitoring by installing the Metrics feature.
The metrics data for a connected device are generated and published:
* automatically - depending on the initial request if the System Metrics application is stared with such configuration (via flag or JSON configuration file)
* on manual request - if metrics request is published as Ditto live inbox message
A currently running metrics data generation can be changed or stopped with a new request.

## Community

* [GitHub Issues](https://github.com/eclipse-kanto/system-metrics/issues)
* [Mailing List](https://accounts.eclipse.org/mailing-list/kanto-dev)
