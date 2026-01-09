#!/usr/bin/env python3
"""
Create Grafana RED metrics dashboards for WMS Platform services.

RED Metrics:
- Rate: Requests per second
- Errors: Error rate (4xx/5xx responses)
- Duration: Request latency percentiles

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


def build_panel(panel_template: dict, service: str, prometheus_uid: str) -> dict:
    """Build a Grafana panel from template."""
    panel = {
        "id": panel_template["id"],
        "title": panel_template["title"],
        "type": panel_template["type"],
        "gridPos": panel_template["gridPos"],
        "datasource": {"type": "prometheus", "uid": prometheus_uid},
        "targets": []
    }

    # Add description if present
    if "description" in panel_template:
        panel["description"] = panel_template["description"]

    # Add fieldConfig if present
    if "fieldConfig" in panel_template:
        panel["fieldConfig"] = panel_template["fieldConfig"]

    # Add options if present
    if "options" in panel_template:
        panel["options"] = panel_template["options"]

    # Add transformations if present
    if "transformations" in panel_template:
        panel["transformations"] = panel_template["transformations"]

    # Build targets (queries)
    for query in panel_template.get("queries", []):
        target = {
            "refId": query["refId"],
            "expr": query["expr"].replace("${SERVICE}", service),
            "datasource": {"type": "prometheus", "uid": prometheus_uid}
        }
        if "legendFormat" in query:
            target["legendFormat"] = query["legendFormat"]
        if "format" in query:
            target["format"] = query["format"]
        if "instant" in query:
            target["instant"] = query["instant"]

        panel["targets"].append(target)

    return panel


def create_dashboard_payload(service: str, config: dict) -> dict:
    """Create Grafana dashboard JSON payload for a service."""
    prometheus_uid = config["prometheus_datasource_uid"]
    template = config["dashboard_template"]

    panels = []
    for panel_template in template["panels"]:
        panel = build_panel(panel_template, service, prometheus_uid)
        panels.append(panel)

    return {
        "dashboard": {
            "title": f"{service}-red-metrics",
            "tags": ["wms", "red-metrics", service],
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
    """Delete existing RED metrics dashboards."""
    # Search for dashboards with 'red-metrics' tag
    search_url = f"{grafana_url}/api/search?tag=red-metrics"
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
        if dashboard.get("title", "").endswith("-red-metrics"):
            uid = dashboard.get("uid")
            if uid:
                delete_url = f"{grafana_url}/api/dashboards/uid/{uid}"
                result = grafana_request(delete_url, "DELETE", None, auth)
                if "error" not in result:
                    deleted += 1
                    print(f"Deleted: {dashboard.get('title')}")

    return deleted


def main():
    parser = argparse.ArgumentParser(description="Create Grafana RED metrics dashboards for WMS Platform")
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
    print("WMS Platform - RED Metrics Dashboard Creator")
    print("=" * 50)
    print(f"Grafana URL: {args.grafana_url}")
    print(f"Prometheus Datasource UID: {config['prometheus_datasource_uid']}")
    print()

    # Optionally clean up existing dashboards
    if args.clean:
        print("Cleaning up existing RED metrics dashboards...")
        deleted = delete_existing_dashboards(args.grafana_url, auth)
        print(f"Deleted {deleted} existing dashboards")
        print()

    # Create dashboards
    print("Creating RED metrics dashboards...")
    success = 0
    failed = 0

    for service in config["services"]:
        payload = create_dashboard_payload(service, config)
        url = f"{args.grafana_url}/api/dashboards/db"
        result = grafana_request(url, "POST", payload, auth)

        if result.get("status") == "success":
            print(f"  Created: {service}-red-metrics")
            success += 1
        else:
            error_msg = result.get('error', result.get('message', 'Unknown error'))
            print(f"  Failed: {service}-red-metrics - {error_msg}")
            failed += 1

    print()
    print("=" * 50)
    print(f"Summary: {success} created, {failed} failed")
    print("=" * 50)
    print()
    print("Dashboard panels include:")
    print("  - Request Rate (req/sec)")
    print("  - Error Rate (errors/sec)")
    print("  - Error Rate (%)")
    print("  - Requests In-Flight")
    print("  - Latency Percentiles (p50, p95, p99)")
    print("  - Latency Heatmap")
    print("  - Top Endpoints by Request Rate")
    print("  - Top Endpoints by Error Rate")
    print()

    return 0 if failed == 0 else 1


if __name__ == "__main__":
    sys.exit(main())
