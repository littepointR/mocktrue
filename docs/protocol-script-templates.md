# Protocol and Script Templates

PortWeave has three related but separate concepts:

1. Visual protocol parser templates: reusable `protocol.FrameSpec` presets for byte-frame parsing.
2. Protocol parser script templates: reusable parser scripts stored in the protocol template registry.
3. Serial graph script nodes: runtime graph nodes that generate, transform, or analyze byte streams with the PortWeave graph script API.

Do not mix the APIs. Protocol parser scripts currently use parser helpers such as `len()` and `byte(i)`. Graph script nodes use `input.bytes()`, `input.hex()`, `input.text()`, `output.bytes()`, `output.hex()`, and `output.text()`.

## When to use each option

Use a visual parser template when the frame shape can be described by fixed pieces:

- optional header bytes
- optional length field
- optional checksum field
- fixed-offset fields
- minimum/maximum frame lengths

Use a protocol parser script template when the protocol is still a parser template but needs logic beyond the current visual model. The current registry includes NMEA as a script parser template. It is not a graph node script and should not use graph `input` or `output` objects.

Use graph script nodes when the graph itself needs to create or process live bytes:

- `serial.script.generator`: emits bytes without an input edge.
- `serial.script.transform`: reads bytes from `input` and writes transformed bytes to `output`.
- `serial.script.analyzer`: reads bytes and records fields/errors without forwarding bytes.

## Current protocol template catalog

Source of truth: `internal/modules/serial/protocol/templates/templates.go`.

| Template | Kind | Purpose | Validates/extracts |
| --- | --- | --- | --- |
| `Modbus RTU` | visual | Standard Modbus RTU frame shape | Slave address, function code, trailing CRC16 |
| `AA55 自定义帧` | visual | Common embedded `AA 55 + length + payload + checksum` frame | Header, little-endian length, `cmd` field, sum8 checksum |
| `NMEA` | script | NMEA 0183 text sentence parser | Sentence text and XOR checksum using parser helpers |

The same registry is exposed through read-only MCP tools:

- `protocol_template_catalog` lists template names, descriptions, and kinds.
- `protocol_template_describe` returns one template's current config by name.

These tools are inspection/export helpers. They do not create, mutate, or register templates at runtime.

## Visual parser model

Visual parser config is `protocol.FrameSpec`:

- `Name`: display/parser name.
- `Header`: fixed byte sequence to search before parsing a frame. Empty means parse from the current offset.
- `LengthField`: offset, width, endianness, and whether the encoded length includes header/checksum bytes.
- `Checksum`: checksum type, offset, width, and algorithm parameters.
- `Fields`: fixed-offset extracted fields with name, offset, width, and display type.
- `MinFrameLen` / `MaxFrameLen`: lower/upper bounds used when a length field is absent or defensive clamping is needed.

Supported checksum algorithms in the visual parser include `none`, `sum8`, `crc8`, `crc16`, and `crc32`.

Example AA55 frame from the current template:

```text
AA 55 04 00 03 01
```

Interpretation:

- `AA 55`: fixed header.
- `04 00`: little-endian length, including checksum.
- `03`: sum8 checksum over bytes before the checksum.
- `01`: payload byte. The template extracts `cmd` from offset 4, which is the checksum byte in the current visual config, so treat the current template as a compatibility fixture rather than a complete product protocol recommendation.

## Protocol parser script templates

The protocol template registry can store `protocol.KindScript` templates. The current NMEA template uses parser-style helpers:

- `len()` returns available byte count.
- `byte(i)` returns the byte at an index.
- `field(name, value, display?)` records parsed fields.
- `error(message)` records parse errors.

These helpers belong to the protocol parser script template context. They are separate from graph script node APIs. Do not copy NMEA's `len()` / `byte(i)` style into `serial.script.*` graph nodes unless that engine explicitly implements the same helpers.

## Graph script node API

Source of truth: `internal/modules/serial/script_runtime.go` and current MCP/frontend provider defaults.

### Node-specific API availability

| Node type | Input API | Output API | Typical use |
| --- | --- | --- | --- |
| `serial.script.generator` | no | yes | Generate startup/interval bytes |
| `serial.script.transform` | yes | yes | Pass through, rewrite, encode/decode, filter-like transforms |
| `serial.script.analyzer` | yes | no | Record fields/errors for evidence |

### Input API

Available to transform and analyzer nodes:

```javascript
input.bytes()          // returns an array of byte numbers
input.hex()            // returns lowercase hex bytes separated by spaces
input.text("utf-8")    // decodes input bytes; encoding defaults to node encoding
```

Supported text encodings are the same serial text encodings used elsewhere in the serial module, including UTF-8, ASCII, UTF-16LE/BE, GBK/GB18030, Big5, Shift-JIS, and Windows code pages covered by `textencoding.go`.

### Output API

Available to generator and transform nodes:

```javascript
output.bytes([0x01, 0x02])
output.hex("01 02")
output.text("tick\n", "utf-8")
```

`output.bytes()` accepts byte arrays or strings. Numeric byte values must be integers in `0..255`.

### Analyzer/control helpers

Available to graph script nodes:

```javascript
field("length", input.bytes().length)
field("temperature_c", 21.5, "21.5 °C")
error("checksum mismatch")
drop()
```

`drop()` suppresses accumulated output. It is most useful in transform scripts; analyzers do not emit output anyway.

### State helpers

Graph scripts can keep bounded JSON-compatible state between runs of the same runtime:

```javascript
var runs = state.get("runs") || 0;
state.set("runs", runs + 1);
state.delete("temporary");
```

State values must be JSON-cloneable and fit under `maxStateBytes`.

### Utility helpers

```javascript
crc16(input.bytes())
sum8(input.bytes())
now()
```

`crc16()` uses the Modbus CRC16 helper. `sum8()` returns the 8-bit byte sum. `now()` returns a nanosecond timestamp.

### Limits and failure behavior

Defaults in current provider metadata/runtime:

- `timeoutMs = 50`
- `maxOutputBytes = 65536`
- `maxStateBytes = 262144`
- `encoding = utf-8`
- `onError = mark-error-and-drop`

Runtime errors include empty scripts, unsupported encodings, invalid hex, invalid byte arrays, output limit overflow, state limit overflow, and execution timeout.

## Graph script examples

### Line generator

```javascript
output.text("tick\n", "utf-8");
```

Use with `serial.script.generator`. It does not have `input`.

### Pass-through transform

```javascript
output.bytes(input.bytes());
```

Use with `serial.script.transform` to verify graph routing without changing payloads.

### Text analyzer with JavaScript string APIs

```javascript
var text = input.text("utf-8");
var match = text.match(/^TEMP=(-?\d+(?:\.\d+)?)$/);
if (match) {
  field("temperature_c", Number(match[1]), match[1] + " °C");
} else {
  error("expected TEMP=<number>");
}
```

Use with `serial.script.analyzer`. JavaScript standard string/regex APIs are available through the script engine; graph-specific parsing still goes through `input.*`, `field()`, and `error()`.

### Binary length/checksum analyzer

```javascript
var bytes = input.bytes();
field("length", bytes.length);
if (bytes.length < 2) {
  error("frame too short");
} else {
  var payload = [];
  for (var i = 0; i < bytes.length - 1; i++) {
    payload.push(bytes[i]);
  }
  var actual = sum8(payload);
  var expected = bytes[bytes.length - 1];
  field("checksum", actual, actual === expected ? "OK" : "mismatch");
  if (actual !== expected) {
    error("checksum mismatch");
  }
}
```

Use with `serial.script.analyzer` for simple binary evidence without changing downstream bytes.

## Troubleshooting

### Script timeout

Reduce loops and per-frame work, or raise `timeoutMs` only when the graph scenario needs it. Keep examples bounded; scripts that run forever fail with `script execution timeout`.

### Output limit exceeded

Check `maxOutputBytes` and avoid generating unbounded output. Prefer writing one frame per run/interval.

### Invalid byte array

Every numeric item passed to `output.bytes()`, `crc16()`, or `sum8()` must be an integer byte in `0..255`. Strings are treated as bytes of the string.

### Missing fields

Analyzer nodes only record fields when scripts call `field()`. A monitor can prove bytes arrived; an analyzer proves the script extracted fields.

### Checksum mismatch

For visual parser templates, verify checksum offset/width and whether the length field includes checksum bytes. For graph analyzer scripts, log both the expected byte and calculated checksum with `field()` before calling `error()`.

## Related docs

- [Serial Graph Node Catalog](serial-graph-node-catalog.md)
- [Serial Graph Runtime Flow](serial-graph-runtime-flow.md)
- [MCP API and Recipes](mcp-api.md)
- [Testing](development/testing.md)
