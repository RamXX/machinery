workspace "PII Flow" "The external-checker reference design: a processing pipeline whose only exit is the export adapter." {
  model {
    sys = softwareSystem "PII Pipeline" "Collects subject data, processes it for stated purposes, redacts before export." {
      pipeline = container "Processing Pipeline" "Subject records, processing activities, redaction." "Go"
      export   = container "Export Adapter" "Ships redacted output to the analytics destination; holds no subject records." "Go"
    }
    analytics = softwareSystem "Analytics Destination" "The reporting/analytics sink outside the processing boundary." "External"
    pipeline -> export "Hands off redacted output only"
    export -> analytics "Delivers redacted exports"
  }
}
