#!/usr/bin/env python3
"""
Deploy Events Platform Overview dashboard to Grafana.

This dashboard provides a comprehensive view of all WMS Kafka event streaming:
- Event throughput (published/consumed)
- Consumer lag monitoring
- Outbox pattern metrics
- Publishing latency

Usage:
    python3 deploy-dashboard.py [--grafana-url URL] [--user USER] [--password PASSWORD]

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


def load_dashboard():
    """Load dashboard JSON from grafana/dashboards directory."""
    dashboard_path = Path(__file__).parent.parent.parent / "grafana" / "dashboards" / "events-platform-overview.json"
    with open(dashboard_path) as f:
        return json.load(f)


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


def delete_existing_dashboard(grafana_url: str, auth: tuple, uid: str) -> bool:
    """Delete existing dashboard by UID."""
    delete_url = f"{grafana_url}/api/dashboards/uid/{uid}"
    result = grafana_request(delete_url, "DELETE", None, auth)
    return "error" not in result


def main():
    parser = argparse.ArgumentParser(description="Deploy Events Platform Overview dashboard to Grafana")
    parser.add_argument("--grafana-url", default=os.environ.get("GRAFANA_URL", "http://localhost:3000"),
                        help="Grafana URL")
    parser.add_argument("--user", default=os.environ.get("GRAFANA_USER", "admin"),
                        help="Grafana username")
    parser.add_argument("--password", default=os.environ.get("GRAFANA_PASSWORD", "admin"),
                        help="Grafana password")
    parser.add_argument("--clean", action="store_true",
                        help="Delete existing dashboard before creating new one")
    args = parser.parse_args()

    auth = (args.user, args.password)

    print("=" * 60)
    print("WMS Platform - Events Platform Overview Dashboard Deployment")
    print("=" * 60)
    print(f"Grafana URL: {args.grafana_url}")
    print()

    # Load dashboard JSON
    try:
        dashboard = load_dashboard()
        print(f"Loaded dashboard: {dashboard.get('title', 'Unknown')}")
        print(f"Dashboard UID: {dashboard.get('uid', 'Unknown')}")
        print(f"Panels: {len(dashboard.get('panels', []))}")
    except FileNotFoundError:
        print("ERROR: Dashboard JSON file not found!")
        print("Expected path: deployments/grafana/dashboards/events-platform-overview.json")
        return 1
    except json.JSONDecodeError as e:
        print(f"ERROR: Invalid JSON in dashboard file: {e}")
        return 1

    print()

    # Optionally delete existing dashboard
    if args.clean:
        print("Cleaning up existing dashboard...")
        uid = dashboard.get("uid", "events-platform-overview")
        if delete_existing_dashboard(args.grafana_url, auth, uid):
            print(f"  Deleted existing dashboard: {uid}")
        else:
            print(f"  No existing dashboard found or could not delete: {uid}")
        print()

    # Deploy dashboard
    print("Deploying Events Platform Overview dashboard...")

    payload = {
        "dashboard": dashboard,
        "overwrite": True,
        "message": "Deployed via deploy-dashboard.py"
    }

    # Reset ID to allow creation
    payload["dashboard"]["id"] = None

    url = f"{args.grafana_url}/api/dashboards/db"
    result = grafana_request(url, "POST", payload, auth)

    if result.get("status") == "success" or result.get("uid"):
        print()
        print("=" * 60)
        print("SUCCESS: Dashboard deployed!")
        print("=" * 60)
        print()
        print(f"Dashboard URL: {args.grafana_url}/d/{result.get('uid', dashboard.get('uid'))}")
        print()
        print("Dashboard features:")
        print("  - Events Health Overview (KPIs)")
        print("  - Event Throughput by Topic & Type")
        print("  - Consumer Lag Analysis (critical for event systems)")
        print("  - Publishing Performance (latency percentiles)")
        print("  - Outbox Pattern Monitoring")
        print("  - Event Flow Balance Analysis")
        print("  - Service-by-Service Events Matrix")
        print()
        print("Template variables available:")
        print("  - Service: Filter by WMS service")
        print("  - Topic: Filter by Kafka topic")
        print("  - Event Type: Filter by CloudEvent type")
        print()
        print("Alerting thresholds (best practices):")
        print("  - Consumer Lag: Warning >1,000, Critical >10,000")
        print("  - Publish Error Rate: Warning >1%, Critical >5%")
        print("  - Outbox Pending: Warning >50, Critical >100")
        print()
        return 0
    else:
        error_msg = result.get('error', result.get('message', 'Unknown error'))
        print()
        print("=" * 60)
        print(f"FAILED: Could not deploy dashboard")
        print("=" * 60)
        print(f"Error: {error_msg}")
        print()
        return 1


if __name__ == "__main__":
    sys.exit(main())
