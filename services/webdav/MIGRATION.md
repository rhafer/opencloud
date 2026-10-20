# Migrating the thumbnail settings

Thumbnails are made by a generator the webdav service posts an image to, and
everything that decides what a thumbnail is now belongs to webdav: the sizes, the
cache, and the types a preview is made of. The thumbnails service keeps only the
limits it can enforce while it holds the image.

Every environment variable that moved is still read under its old name, so a
deployment configured by environment keeps working. A yaml configuration does
not: a key belongs to the service section it sits in, and a key nobody knows is
ignored without a word, which leaves the default in place.

## v8.x.x to v9.0.0

### A generator has to be reachable

The thumbnails service used to be resolved through the service registry. webdav
now posts to a URL, `http://127.0.0.1:9186` by default, so a deployment that runs
the services in separate containers has to say where the generator is:

```shell
WEBDAV_THUMBNAILS_GENERATOR_URL=http://thumbnails:9186
```

Nothing has to be set in the single binary. The URL may also point at a real
[imagor](https://imagor.net), which speaks the same endpoint.

### Settings that changed service

| Old name | Now | |
| --- | --- | --- |
| `THUMBNAILS_RESOLUTIONS` | `WEBDAV_THUMBNAILS_RESOLUTIONS` | a ladder of edge lengths, not a list of boxes |
| `THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE` | `OC_THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE` | read by webdav and the generator, which have to agree |
| `THUMBNAILS_TXT_FONTMAP_FILE` | `WEBDAV_THUMBNAILS_TXT_FONTMAP_FILE` | webdav renders text before it posts |
| `THUMBNAILS_FILESYSTEMSTORAGE_ROOT` | `WEBDAV_THUMBNAILS_CACHE_DIR` | the old name is the directory that holds `files/`, the new one is the directory itself |

The thumbnail cache has to be reachable by the webdav container after the move.

### Settings that are gone

`THUMBNAILS_DATA_ENDPOINT`, `THUMBNAILS_TRANSFER_TOKEN` and
`THUMBNAILS_WEBDAVSOURCE_INSECURE` configured the pipeline that fetched the file
itself. The generator is handed the image now, so there is nothing to point them
at.

### A yaml configuration has to move

The keys live under the section of the service that reads them, so they move with
the settings. Before:

```yaml
thumbnails:
  thumbnail:
    resolutions: ["16x16", "32x32", "7680x4320"]
    max_input_image_file_size: 50MB
    font_map_file: /etc/opencloud/fontmap.json
  filesystem_storage:
    root: /var/lib/opencloud/thumbnails
```

After:

```yaml
webdav:
  thumbnails_resolutions: ["16", "32", "2048"]
  thumbnails_max_input_image_file_size: 50MB
  thumbnails_txt_fontmap_file: /etc/opencloud/fontmap.json
  thumbnails_cache_dir: /var/lib/opencloud/thumbnails/files
thumbnails:
  thumbnail:
    max_input_image_file_size: 50MB
```

### What changes without any configuration

The resolutions are edge lengths now, and each axis of a request is rounded up on
its own, so a preview is no longer returned in a different orientation than it was
asked for. The default ladder ends at 2048, where it used to reach 8K, so a client
that asks for more gets 2048.

A source smaller than the box is enlarged to fill it for the cropping operations,
the way imagor's default resize does, where the previous service returned the
source size instead. `fit-in` still never enlarges.

The cache key covers the snapped box, the operation, the output format and the
file's type, so the cache is cold after the update and fills again as previews are
requested. Nothing has to be deleted.

Which types a preview is announced and attempted for is one list for webdav, graph
and the PROPFIND property, set through `OC_THUMBNAILS_SOURCE_TYPES`. A type the
running generator cannot read answers `404`.
