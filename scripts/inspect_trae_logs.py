#!/usr/bin/env python3
"""Summarize local Trae protocol evidence without exporting log values.

Reads plaintext .log files only. Output uses fixed protocol allowlists: no raw
lines, prompts, token values, request IDs, device IDs, or account identifiers.
This is an offline research utility; it does not authenticate or send requests.
"""

import argparse
from collections import Counter
import hashlib
import json
from pathlib import Path
import re


ROUTES = (
    "/api/ide/v1/batch_get_detail_param",
    "/api/ide/v1/get_detail_param",
    "/api/ide/v1/get_model_list",
    "/api/ide/v1/model_list",
    "/api/ide/v1/llm_raw_chat",
    "/api/ide/v2/llm_raw_chat",
    "/api/agent/v3/llm_utils_chat",
    "/api/agent/v3/workflow/start",
)
HOSTS = ("trae-api-cn.mchost.guru", "api5-normal.mchost.guru", "api.trae.com.cn", "api.trae.cn")
HEADERS = frozenset(
    "authorization x-cloudide-token x-ide-token x-uid x-app-id x-app-version "
    "x-app-version-code x-custom-trace-id x-device-brand x-device-cpu "
    "x-device-id x-device-type x-flow-traceparent x-ide-version "
    "x-ide-version-code x-ide-version-type x-machine-id x-os-version "
    "request-traffic-type x-request-pin x-requested-at".split()
)
FUNCTIONS = frozenset(
    "assistant solo_agent solo_agent_lite solo_coder solo_agent_remote "
    "solo_work_lite solo_work_remote solo_design_lite solo_design_remote builder".split()
)
MODEL_FIELDS = frozenset(
    "provider is_preset config_name config_source model_name display_model_name "
    "ak base_url use_remote_service multimodal support_read_image prompt_max_tokens "
    "toolcall_history_max_tokens encrypted_model_params extra_config "
    "function_extra_config ab_versions persist_meta raw_chat_function prompt_set "
    "context_window_sizes max_turn display_options max_tokens application_config "
    "sk auth_type region session_token user_custom_hyper_params custom_model_type "
    "reasoning_effort".split()
)
EVENTS = ("TimingCostOriginEvent", "UnknownEvent", "CloudContextUsageEvent")
URL = re.compile(r"https://([a-z0-9.-]+)(/api/[a-z0-9_./-]+)")
QUOTED = re.compile(r'"(?:\\.|[^"\\])*"')


def summarize(root):
    endpoints = {}
    event_counts = Counter()
    headers = set()
    versions = Counter()
    model_fields = set()
    functions = set()
    evidence = []
    scanned = skipped = 0
    for path in sorted(root.rglob("*.log")):
        if not path.is_file():
            continue
        if path.stat().st_size > 30_000_000:
            skipped += 1
            continue
        data = path.read_bytes()
        scanned += 1
        matches = []
        for number, line in enumerate(data.decode("utf-8", errors="replace").splitlines(), 1):
            for host, route in URL.findall(line):
                if host not in HOSTS or route not in ROUTES:
                    continue
                key = "https://" + host + route
                item = endpoints.setdefault(key, {"mentions": 0, "send_records": 0, "methods": {}, "response_statuses": {}, "contexts": {}})
                item["mentions"] += 1
                if "send: calling Fetch" in line:
                    item["send_records"] += 1
                    method = re.search(r"method=(GET|POST|PUT|DELETE|PATCH),", line)
                    if method:
                        name = method.group(1)
                        item["methods"][name] = item["methods"].get(name, 0) + 1
                if "[AhaNetHTTPClient/Stream]" in line:
                    status = re.search(r"Status: ([1-5][0-9]{2})\b", line)
                    if status:
                        name = status.group(1)
                        item["response_statuses"][name] = item["response_statuses"].get(name, 0) + 1
                if "generate_session_title_and_icon:" in line:
                    name = "generate_session_title_and_icon"
                    item["contexts"][name] = item["contexts"].get(name, 0) + 1
                if len(matches) < 12:
                    matches.append({"line": number, "kind": "endpoint", "endpoint": key})
            for name in EVENTS:
                if name + " {" in line:
                    event_counts[name] += 1
            if "[HTTPClient] add_header {" in line:
                try:
                    obj, _ = json.JSONDecoder().raw_decode(line.split("add_header ", 1)[1])
                    if isinstance(obj, dict):
                        headers.update(str(k).lower() for k in obj if str(k).lower() in HEADERS)
                        version = obj.get("x-ide-version", "")
                        code = obj.get("x-ide-version-code", "")
                        if isinstance(version, str) and isinstance(code, str) and re.fullmatch(r"[0-9.]{1,24}", version) and re.fullmatch(r"[0-9]{1,12}", code):
                            versions[(version, code)] += 1
                except (ValueError, TypeError):
                    pass
            if "BatchDetailParamRequest {" in line:
                # Restrict values to known function enums; arbitrary strings never leave the process.
                match = re.search(r"functions: \[([^\]]*)\]", line)
                if match:
                    functions.update(v for v in re.findall(r'"([a-z_]+)"', match.group(1)) if v in FUNCTIONS)
            if "model_info: CustomModel {" in line:
                # Strip all quoted data before identifying Rust struct field names.
                shape = QUOTED.sub('""', line.split("model_info: CustomModel {", 1)[1])
                model_fields.update(k for k in re.findall(r"\b([a-z_]+):", shape) if k in MODEL_FIELDS)
                if len(matches) < 12:
                    matches.append({"line": number, "kind": "custom_model_shape"})
        if matches:
            evidence.append({"file": str(path.relative_to(root)), "sha256": hashlib.sha256(data).hexdigest(), "locations": matches})
    return {
        "scope": "historical plaintext logs only; counts are log mentions, not unique requests or current capabilities",
        "privacy": "fixed allowlisted metadata only; no original lines or credential values exported",
        "scanned_files": scanned,
        "skipped_large_files": skipped,
        "endpoints": endpoints,
        "header_names_observed_in_add_header": sorted(headers),
        "version_header_mentions": [{"ide_version": k[0], "ide_version_code": k[1], "mentions": v} for k, v in sorted(versions.items())],
        "batch_detail_functions_observed": sorted(functions),
        "custom_model_field_names": sorted(model_fields),
        "event_debug_type_mentions": dict(event_counts),
        "evidence": evidence,
    }


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("log_root", type=Path)
    args = parser.parse_args()
    if not args.log_root.is_dir():
        parser.error("log_root must be an existing directory")
    print(json.dumps(summarize(args.log_root), ensure_ascii=False, indent=2))
