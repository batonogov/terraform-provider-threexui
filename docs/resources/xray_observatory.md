---
page_title: "threexui_xray_observatory Resource - 3x-ui"
subcategory: "Xray Settings"
description: |-
  Manages Xray Observatory and BurstObservatory configuration in the 3x-ui panel.
---

# threexui_xray_observatory (Resource)

Manages Xray Observatory and BurstObservatory configuration on the 3x-ui panel (3x-ui v3.4.2+). The observatories produce outbound latency data used by balancer strategies such as `leastPing` and `leastLoad`.

The resource owns both top-level Xray template keys: `observatory` and `burstObservatory`. On apply, each configured section replaces the corresponding key, while unrelated Xray template keys are preserved through read-modify-write. Removing all blocks for one section removes that key from the template. Deleting the Terraform resource only removes it from state; it does not clear either panel setting.

## Example Usage

```terraform
resource "threexui_xray_observatory" "obs" {
  observatory {
    tag                = "default"
    subject_selector   = ["proxy-*"]
    probe_url          = "https://www.google.com/generate_204"
    probe_interval     = "1m"
    enable_concurrency = true
  }

  burst_observatory {
    tag              = "default"
    subject_selector = ["proxy-*"]

    ping_config {
      destination     = "https://www.cloudflare.com/cdn-cgi/trace"
      interval        = "1m"
      connect_timeout = "5s"
      timeout         = "10s"
      samples         = 3
      sampling_count  = 2
      lazy            = true
    }
  }
}
```

## Notes

- Xray-core (v26.6.27+) reads both sections as single flat objects. Use
  `tag = "default"` for the entry Xray should apply; other tags round-trip
  through Terraform state for migration but are not read by Xray.
- `burst_observatory` must include `ping_config`: Xray-core refuses to start
  with a burst observatory that has no ping config ("BurstObservatory requires
  a valid pingConfig").

## Argument Reference

### observatory (Optional, Block List)

- `tag` (Required, String) - Use `default` for a single flat Xray observatory.
- `subject_selector` (Optional+Computed, List of String) - Outbound tag prefixes or patterns to probe.
- `probe_url` (Optional, String) - URL to probe for latency.
- `probe_interval` (Optional, String) - Probe interval (e.g. `1m`, `30s`).
- `enable_concurrency` (Optional+Computed, Boolean) - Probe all matching outbounds concurrently.

### burst_observatory (Optional, Block List)

- `tag` (Required, String) - Use `default` for a single flat Xray burst observatory.
- `subject_selector` (Optional+Computed, List of String) - Outbound tag prefixes or patterns to probe.
- `ping_config` (Optional, Block List, Max: 1) - Burst probing configuration.

#### ping_config (Optional, Block List, Max: 1)

- `destination` (Optional+Computed, String) - Probe destination URL.
- `interval` (Optional+Computed, String) - Interval between probes (e.g. `1m`).
- `connect_timeout` (Optional+Computed, String) - Connection timeout for each probe.
- `timeout` (Optional+Computed, String) - Overall probe timeout.
- `samples` (Optional+Computed, Number) - Number of samples per probe.
- `sampling_count` (Optional+Computed, Number) - Number of sampling rounds per probe.
- `lazy` (Optional+Computed, Boolean) - Probe only when a connection request is received.

## Read-Only Attributes

- `id` (String) - Singleton resource ID, always `xray_observatory`.

## Import

Import the singleton using its canonical ID:

```shell
terraform import threexui_xray_observatory.obs xray_observatory
```

Import reads both top-level observatory keys from the panel into Terraform state.
