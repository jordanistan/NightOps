# Embedded target catalog

`targets-v1.csv` is a small, versioned offline seed catalog. Coordinates are
J2000-equivalent decimal degrees transcribed from NASA Hubble object pages:

- [Andromeda Galaxy M31](https://science.nasa.gov/asset/hubble/andromeda-galaxy-m31/)
- [Evolution of the Orion Nebula M42](https://science.nasa.gov/asset/hubble/evolution-of-the-orion-nebula-m42/)
- [Compass and Scale Image of M13](https://science.nasa.gov/asset/hubble/compass-and-scale-image-of-m13/)

This catalog is for planning and identification, not a complete astronomical
database. Future SkyBase imports must preserve source and coordinate epoch.

The CSV is embedded into the binary and loaded at startup for offline target
selection. Selected targets are copied into the mission database by stable ID;
live Wikipedia-compatible reference summaries are cached separately and are
never required to start an observing session.
