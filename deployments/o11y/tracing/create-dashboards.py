#!/usr/bin/env python3
"""
Create Grafana tracing dashboards for WMS Platform services.

Usage:
    python3 create-dashboards.py [--grafana-url URL] [--user USER] [--password PASSWORD] [--clean]

Environment variables:
    GRAFANA_URL: Grafana URL (default: http://localhost:3000)
    GRAFANA_USER: Grafana username (default: admin)
    GRAFANA_PASSWORD: Grafana password (default: admin)
"""

import json
import os
import sys
import urllib.request
import base64
import argparse
from pathlib import Path


def load_config():
    """Load dashboard configuration from dashboards.json."""
    config_path = Path(__file__).parent / "dashboards.json"
    with open(config_path) as f:
        return json.load(f)


def create_dashboard_payload(service: str, config: dict) -> dict:
    """Create Grafana dashboard JSON payload for a service."""
    tempo_uid = config["tempo_datasource_uid"]
    loki_uid = config["loki_datasource_uid"]
    namespace = config["namespace"]
    template = config["dashboard_template"]

    panels = [
        # Panel 1: Recent Traces Table
        {
            "id": 1,
            "title": "Recent Traces",
            "type": "table",
            "gridPos": {"x": 0, "y": 0, "w": 24, "h": 10},
            "description": f"Recent traces for {service}",
            "datasource": {"type": "tempo", "uid": tempo_uid},
            "targets": [
                {
                    "refId": "A",
                    "queryType": "traceqlSearch",
                    "limit": 50,
                    "tableType": "traces",
                    "filters": [
                        {
                            "id": "service-name",
                            "tag": "service.name",
                            "operator": "=",
                            "value": [service],
                            "valueType": "string",
                            "scope": "resource"
                        }
                    ]
                }
            ],
            "options": {
                "showHeader": True,
                "cellHeight": "sm"
            },
            "fieldConfig": {
                "defaults": {
                    "custom": {
                        "align": "auto",
                        "cellOptions": {"type": "auto"},
                        "inspect": False
                    }
                },
                "overrides": [
                    {
                        "matcher": {"id": "byName", "options": "Trace ID"},
                        "properties": [
                            {"id": "links", "value": [
                                {
                                    "title": "View Trace",
                                    "url": "/explore?orgId=1&left=%7B%22datasource%22:%22tempo%22,%22queries%22:%5B%7B%22refId%22:%22A%22,%22datasource%22:%7B%22type%22:%22tempo%22,%22uid%22:%22tempo%22%7D,%22queryType%22:%22traceql%22,%22limit%22:20,%22query%22:%22${__value.raw}%22%7D%5D%7D",
                                    "targetBlank": True
                                }
                            ]}
                        ]
                    }
                ]
            }
        },
        # Panel 2: Trace Duration Histogram
        {
            "id": 2,
            "title": "Trace Duration Distribution",
            "type": "histogram",
            "gridPos": {"x": 0, "y": 10, "w": 12, "h": 8},
            "description": "Distribution of trace durations",
            "datasource": {"type": "tempo", "uid": tempo_uid},
            "targets": [
                {
                    "refId": "A",
                    "queryType": "traceqlSearch",
                    "limit": 100,
                    "tableType": "traces",
                    "filters": [
                        {
                            "id": "service-name",
                            "tag": "service.name",
                            "operator": "=",
                            "value": [service],
                            "valueType": "string",
                            "scope": "resource"
                        }
                    ]
                }
            ],
            "fieldConfig": {
                "defaults": {
                    "unit": "ms"
                }
            },
            "options": {
                "legend": {"displayMode": "list", "placement": "bottom"},
                "bucketSize": 50,
                "combine": False
            },
            "transformations": [
                {
                    "id": "filterFieldsByName",
                    "options": {"include": {"names": ["Duration"]}}
                }
            ]
        },
        # Panel 3: Traces Over Time
        {
            "id": 3,
            "title": "Traces Over Time",
            "type": "timeseries",
            "gridPos": {"x": 12, "y": 10, "w": 12, "h": 8},
            "description": "Number of traces over time",
            "datasource": {"type": "tempo", "uid": tempo_uid},
            "targets": [
                {
                    "refId": "A",
                    "queryType": "traceqlSearch",
                    "limit": 500,
                    "tableType": "traces",
                    "filters": [
                        {
                            "id": "service-name",
                            "tag": "service.name",
                            "operator": "=",
                            "value": [service],
                            "valueType": "string",
                            "scope": "resource"
                        }
                    ]
                }
            ],
            "fieldConfig": {
                "defaults": {
                    "unit": "short",
                    "color": {"mode": "palette-classic"}
                }
            },
            "options": {
                "legend": {"displayMode": "list", "placement": "bottom"}
            },
            "transformations": [
                {
                    "id": "filterFieldsByName",
                    "options": {"include": {"names": ["Start time"]}}
                },
                {
                    "id": "groupBy",
                    "options": {
                        "fields": {
                            "Start time": {
                                "aggregations": ["count"],
                                "operation": "groupby"
                            }
                        }
                    }
                }
            ]
        },
        # Panel 4: Error Traces
        {
            "id": 4,
            "title": "Error Traces",
            "type": "table",
            "gridPos": {"x": 0, "y": 18, "w": 24, "h": 8},
            "description": "Traces with errors",
            "datasource": {"type": "tempo", "uid": tempo_uid},
            "targets": [
                {
                    "refId": "A",
                    "queryType": "traceql",
                    "limit": 50,
                    "tableType": "traces",
                    "query": f'{{resource.service.name = "{service}" && status = error}}'
                }
            ],
            "options": {
                "showHeader": True,
                "cellHeight": "sm"
            },
            "fieldConfig": {
                "defaults": {
                    "custom": {
                        "align": "auto",
                        "cellOptions": {"type": "auto"}
                    }
                }
            }
        },
        # Panel 5: Correlated Logs
        {
            "id": 5,
            "title": "Service Logs (with Trace Correlation)",
            "type": "logs",
            "gridPos": {"x": 0, "y": 26, "w": 24, "h": 10},
            "description": "Logs from this service with trace IDs for correlation",
            "datasource": {"type": "loki", "uid": loki_uid},
            "targets": [
                {
                    "refId": "A",
                    "expr": f'{{namespace="{namespace}", pod=~"{service}.*"}} | json | trace_id != ""',
                    "queryType": "range"
                }
            ],
            "options": {
                "showTime": True,
                "showLabels": True,
                "showCommonLabels": False,
                "wrapLogMessage": True,
                "prettifyLogMessage": True,
                "enableLogDetails": True,
                "dedupStrategy": "none",
                "sortOrder": "Descending"
            }
        }
    ]

    return {
        "dashboard": {
            "title": f"{service}-tracing",
            "tags": ["wms", "tracing", service],
            "timezone": template["timezone"],
            "schemaVersion": template["schemaVersion"],
            "version": 0,
            "refresh": template["refresh"],
            "time": template["time"],
            "panels": panels,
            "annotations": {
                "list": [
                    {
                        "builtIn": 1,
                        "datasource": {"type": "grafana", "uid": "-- Grafana --"},
                        "enable": True,
                        "hide": True,
                        "iconColor": "rgba(0, 211, 255, 1)",
                        "name": "Annotations & Alerts",
                        "type": "dashboard"
                    }
                ]
            }
        },
        "overwrite": True
    }


def grafana_request(url: str, method: str, data: dict, auth: tuple) -> dict:
    """Make authenticated request to Grafana API."""
    credentials = base64.b64encode(f"{auth[0]}:{auth[1]}".encode()).decode()

    req = urllib.request.Request(
        url,
        data=json.dumps(data).encode('utf-8') if data else None,
        headers={
            "Content-Type": "application/json",
            "Authorization": f"Basic {credentials}"
        },
        method=method
    )

    try:
        with urllib.request.urlopen(req) as response:
            return json.loads(response.read().decode())
    except urllib.error.HTTPError as e:
        error_body = e.read().decode() if e.fp else str(e)
        return {"error": error_body, "status": e.code}


def delete_existing_dashboards(grafana_url: str, auth: tuple) -> int:
    """Delete existing tracing dashboards."""
    search_url = f"{grafana_url}/api/search?tag=tracing"
    credentials = base64.b64encode(f"{auth[0]}:{auth[1]}".encode()).decode()

    req = urllib.request.Request(
        search_url,
        headers={"Authorization": f"Basic {credentials}"},
        method="GET"
    )

    try:
        with urllib.request.urlopen(req) as response:
            dashboards = json.loads(response.read().decode())
    except Exception as e:
        print(f"Warning: Could not search existing dashboards: {e}")
        return 0

    deleted = 0
    for dashboard in dashboards:
        if dashboard.get("title", "").endswith("-tracing"):
            uid = dashboard.get("uid")
            if uid:
                delete_url = f"{grafana_url}/api/dashboards/uid/{uid}"
                result = grafana_request(delete_url, "DELETE", None, auth)
                if "error" not in result:
                    deleted += 1
                    print(f"Deleted: {dashboard.get('title')}")

    return deleted


def main():
    parser = argparse.ArgumentParser(description="Create Grafana tracing dashboards for WMS Platform")
    parser.add_argument("--grafana-url", default=os.environ.get("GRAFANA_URL", "http://localhost:3000"),
                        help="Grafana URL")
    parser.add_argument("--user", default=os.environ.get("GRAFANA_USER", "admin"),
                        help="Grafana username")
    parser.add_argument("--password", default=os.environ.get("GRAFANA_PASSWORD", "admin"),
                        help="Grafana password")
    parser.add_argument("--clean", action="store_true",
                        help="Delete existing dashboards before creating new ones")
    args = parser.parse_args()

    auth = (args.user, args.password)
    config = load_config()

    print("=" * 50)
    print("WMS Platform - Tracing Dashboard Creator")
    print("=" * 50)
    print(f"Grafana URL: {args.grafana_url}")
    print(f"Tempo Datasource UID: {config['tempo_datasource_uid']}")
    print(f"Loki Datasource UID: {config['loki_datasource_uid']}")
    print()

    # Optionally clean up existing dashboards
    if args.clean:
        print("Cleaning up existing tracing dashboards...")
        deleted = delete_existing_dashboards(args.grafana_url, auth)
        print(f"Deleted {deleted} existing dashboards")
        print()

    # Create dashboards
    print("Creating tracing dashboards...")
    success = 0
    failed = 0

    for service in config["services"]:
        payload = create_dashboard_payload(service, config)
        url = f"{args.grafana_url}/api/dashboards/db"
        result = grafana_request(url, "POST", payload, auth)

        if result.get("status") == "success":
            print(f"  Created: {service}-tracing")
            success += 1
        else:
            error_msg = result.get('error', result.get('message', 'Unknown error'))
            print(f"  Failed: {service}-tracing - {error_msg}")
            failed += 1

    print()
    print("=" * 50)
    print(f"Summary: {success} created, {failed} failed")
    print("=" * 50)
    print()
    print("Dashboard panels include:")
    print("  - Recent Traces (with links to explore)")
    print("  - Trace Duration Distribution")
    print("  - Traces Over Time")
    print("  - Error Traces")
    print("  - Correlated Logs (with trace IDs)")
    print()

    return 0 if failed == 0 else 1


if __name__ == "__main__":
    sys.exit(main())
