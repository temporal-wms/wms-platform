#!/usr/bin/env python3
"""
WMS Platform - Superset Dashboard Setup Script
Creates datasets, charts, and dashboards for:
1. Orders by Requirements Bar Chart (on Orders Dashboard)
2. Order Flow Tracker Dashboard
"""

import requests
import json
import sys

# Superset configuration
SUPERSET_URL = "http://localhost:8088"
USERNAME = "admin"
PASSWORD = "admin"

# Colors
GREEN = "\033[92m"
RED = "\033[91m"
YELLOW = "\033[93m"
BLUE = "\033[94m"
NC = "\033[0m"


class SupersetClient:
    def __init__(self, base_url, username, password):
        self.base_url = base_url
        self.session = requests.Session()
        self.access_token = None
        self.csrf_token = None
        self._login(username, password)

    def _login(self, username, password):
        """Login and get access token"""
        print(f"{YELLOW}Authenticating with Superset...{NC}")

        # Get JWT token
        response = self.session.post(
            f"{self.base_url}/api/v1/security/login",
            json={
                "username": username,
                "password": password,
                "provider": "db",
                "refresh": True
            }
        )

        if response.status_code != 200:
            print(f"{RED}Failed to authenticate: {response.text}{NC}")
            sys.exit(1)

        self.access_token = response.json().get("access_token")
        self.session.headers.update({
            "Authorization": f"Bearer {self.access_token}",
            "Content-Type": "application/json"
        })

        # Get CSRF token
        response = self.session.get(f"{self.base_url}/api/v1/security/csrf_token/")
        if response.status_code == 200:
            self.csrf_token = response.json().get("result")
            self.session.headers.update({"X-CSRFToken": self.csrf_token})

        print(f"{GREEN}✓ Authenticated successfully{NC}")

    def get(self, endpoint, params=None):
        return self.session.get(f"{self.base_url}{endpoint}", params=params)

    def post(self, endpoint, data):
        # Refresh CSRF token before POST
        response = self.session.get(f"{self.base_url}/api/v1/security/csrf_token/")
        if response.status_code == 200:
            self.csrf_token = response.json().get("result")
            self.session.headers.update({"X-CSRFToken": self.csrf_token})
        return self.session.post(f"{self.base_url}{endpoint}", json=data)

    def put(self, endpoint, data):
        response = self.session.get(f"{self.base_url}/api/v1/security/csrf_token/")
        if response.status_code == 200:
            self.csrf_token = response.json().get("result")
            self.session.headers.update({"X-CSRFToken": self.csrf_token})
        return self.session.put(f"{self.base_url}{endpoint}", json=data)


def print_banner():
    print(f"{BLUE}")
    print("╔═══════════════════════════════════════════════════════════╗")
    print("║         WMS Platform - Superset Dashboard Setup           ║")
    print("╚═══════════════════════════════════════════════════════════╝")
    print(f"{NC}")


def create_database_connection(client):
    """Create Trino database connection if not exists"""
    print(f"\n{YELLOW}Setting up Trino database connection...{NC}")

    # Check if Trino connection exists
    response = client.get("/api/v1/database/")
    databases = response.json().get("result", [])

    for db in databases:
        if "trino" in db.get("database_name", "").lower() or "wms" in db.get("database_name", "").lower():
            print(f"{GREEN}✓ Database connection already exists: {db['database_name']} (ID: {db['id']}){NC}")
            return db["id"]

    # Create new connection
    response = client.post("/api/v1/database/", {
        "database_name": "Trino - WMS Data Mesh",
        "engine": "trino",
        "sqlalchemy_uri": "trino://trino@trino.data-mesh.svc.cluster.local:8080/iceberg",
        "expose_in_sqllab": True,
        "allow_ctas": False,
        "allow_cvas": False,
        "allow_dml": False,
        "allow_run_async": True,
        "extra": json.dumps({
            "metadata_params": {},
            "engine_params": {},
            "metadata_cache_timeout": {},
            "schemas_allowed_for_file_upload": []
        })
    })

    if response.status_code in [200, 201]:
        db_id = response.json().get("id")
        print(f"{GREEN}✓ Created Trino database connection (ID: {db_id}){NC}")
        return db_id
    else:
        print(f"{YELLOW}Could not create database connection: {response.status_code}{NC}")
        print(f"{YELLOW}Please create it manually in Superset UI{NC}")
        # Return first available database as fallback
        if databases:
            return databases[0]["id"]
        return None


def create_virtual_dataset(client, database_id, name, sql, schema=None):
    """Create a virtual SQL dataset in Superset"""
    print(f"  Creating virtual dataset: {name}...", end=" ", flush=True)

    # Check if dataset exists
    response = client.get("/api/v1/dataset/", params={"q": f"(filters:!((col:table_name,opr:eq,value:'{name}')))"})
    datasets = response.json().get("result", [])

    for ds in datasets:
        if ds.get("table_name") == name:
            print(f"{GREEN}exists (ID: {ds['id']}){NC}")
            return ds["id"]

    # Create virtual dataset
    payload = {
        "database": database_id,
        "table_name": name,
        "sql": sql
    }
    if schema:
        payload["schema"] = schema

    response = client.post("/api/v1/dataset/", payload)

    if response.status_code in [200, 201]:
        ds_id = response.json().get("id")
        print(f"{GREEN}created (ID: {ds_id}){NC}")
        return ds_id
    else:
        # Try alternative approach
        print(f"{YELLOW}trying alternative...{NC}", end=" ", flush=True)
        payload = {
            "database": database_id,
            "table_name": name,
            "sql": sql,
            "schema": "gold" if not schema else schema
        }
        response = client.post("/api/v1/dataset/", payload)
        if response.status_code in [200, 201]:
            ds_id = response.json().get("id")
            print(f"{GREEN}created (ID: {ds_id}){NC}")
            return ds_id
        print(f"{RED}failed - {response.status_code}: {response.text[:100]}{NC}")
        return None


def create_dataset(client, database_id, schema, table_name, description):
    """Create a dataset in Superset"""
    print(f"  Creating dataset: {schema}.{table_name}...", end=" ", flush=True)

    # Check if dataset exists
    response = client.get("/api/v1/dataset/")
    datasets = response.json().get("result", [])

    for ds in datasets:
        if ds.get("table_name") == table_name:
            print(f"{GREEN}exists (ID: {ds['id']}){NC}")
            return ds["id"]

    response = client.post("/api/v1/dataset/", {
        "database": database_id,
        "schema": schema,
        "table_name": table_name,
        "description": description
    })

    if response.status_code in [200, 201]:
        ds_id = response.json().get("id")
        print(f"{GREEN}created (ID: {ds_id}){NC}")
        return ds_id
    else:
        print(f"{YELLOW}table not found, trying SQL dataset...{NC}")
        return None


def create_sql_dataset(client, database_id, name, sql, description):
    """Create a virtual SQL dataset in Superset"""
    print(f"  Creating SQL dataset: {name}...", end=" ", flush=True)

    # Check if dataset exists
    response = client.get("/api/v1/dataset/")
    datasets = response.json().get("result", [])

    for ds in datasets:
        if ds.get("table_name") == name:
            print(f"{GREEN}exists (ID: {ds['id']}){NC}")
            return ds["id"]

    response = client.post("/api/v1/dataset/", {
        "database": database_id,
        "table_name": name,
        "sql": sql,
        "description": description
    })

    if response.status_code in [200, 201]:
        ds_id = response.json().get("id")
        print(f"{GREEN}created (ID: {ds_id}){NC}")
        return ds_id
    else:
        print(f"{RED}failed - {response.status_code}{NC}")
        return None


def create_chart(client, name, viz_type, datasource_id, params, description=""):
    """Create a chart in Superset"""
    print(f"  Creating chart: {name}...", end=" ", flush=True)

    # Check if chart exists
    response = client.get("/api/v1/chart/")
    charts = response.json().get("result", [])

    for chart in charts:
        if chart.get("slice_name") == name:
            print(f"{GREEN}exists (ID: {chart['id']}){NC}")
            return chart["id"]

    response = client.post("/api/v1/chart/", {
        "slice_name": name,
        "viz_type": viz_type,
        "datasource_id": datasource_id,
        "datasource_type": "table",
        "params": json.dumps(params),
        "description": description
    })

    if response.status_code in [200, 201]:
        chart_id = response.json().get("id")
        print(f"{GREEN}created (ID: {chart_id}){NC}")
        return chart_id
    else:
        print(f"{RED}failed - {response.status_code}{NC}")
        return None


def create_dashboard(client, name, slug, chart_ids, description=""):
    """Create a dashboard in Superset"""
    print(f"  Creating dashboard: {name}...", end=" ", flush=True)

    # Check if dashboard exists
    response = client.get("/api/v1/dashboard/")
    dashboards = response.json().get("result", [])

    for dash in dashboards:
        if dash.get("slug") == slug or dash.get("dashboard_title") == name:
            print(f"{GREEN}exists (ID: {dash['id']}){NC}")
            return dash["id"]

    # Build position JSON for layout
    position_json = {
        "DASHBOARD_VERSION_KEY": "v2",
        "ROOT_ID": {"type": "ROOT", "id": "ROOT_ID", "children": ["GRID_ID"]},
        "GRID_ID": {"type": "GRID", "id": "GRID_ID", "children": ["ROW-1"], "parents": ["ROOT_ID"]},
        "HEADER_ID": {"id": "HEADER_ID", "type": "HEADER", "meta": {"text": name}},
        "ROW-1": {
            "type": "ROW",
            "id": "ROW-1",
            "children": [],
            "parents": ["ROOT_ID", "GRID_ID"],
            "meta": {"background": "BACKGROUND_TRANSPARENT"}
        }
    }

    # Add charts to layout
    for i, chart_id in enumerate(chart_ids):
        if chart_id:
            chart_key = f"CHART-{chart_id}"
            position_json["ROW-1"]["children"].append(chart_key)
            position_json[chart_key] = {
                "type": "CHART",
                "id": chart_key,
                "children": [],
                "parents": ["ROOT_ID", "GRID_ID", "ROW-1"],
                "meta": {"width": 4, "height": 50, "chartId": chart_id}
            }

    response = client.post("/api/v1/dashboard/", {
        "dashboard_title": name,
        "slug": slug,
        "published": True,
        "position_json": json.dumps(position_json),
        "json_metadata": json.dumps({
            "timed_refresh_immune_slices": [],
            "expanded_slices": {},
            "refresh_frequency": 0,
            "default_filters": "{}",
            "color_scheme": "supersetColors"
        })
    })

    if response.status_code in [200, 201]:
        dash_id = response.json().get("id")
        print(f"{GREEN}created (ID: {dash_id}){NC}")
        return dash_id
    else:
        print(f"{RED}failed - {response.status_code}{NC}")
        return None


def setup_orders_requirements_chart(client, database_id):
    """Setup the Orders by Requirements bar chart"""
    print(f"\n{BLUE}═══ Setting up Orders by Requirements Chart ═══{NC}")

    # Try physical table first, fallback to virtual dataset
    dataset_id = create_dataset(
        client, database_id,
        "gold", "orders_by_requirements_daily",
        "Daily aggregation of orders by special requirements"
    )

    if not dataset_id:
        # Create virtual dataset with sample/placeholder SQL
        sql = """
SELECT
    'gift_wrap' as requirement,
    COUNT(*) as order_count,
    100.0 * COUNT(*) / SUM(COUNT(*)) OVER() as percentage_of_total
FROM iceberg.bronze.orders_raw
WHERE gift_wrap = true
GROUP BY 1
UNION ALL
SELECT
    'multi_item' as requirement,
    COUNT(*) as order_count,
    100.0 * COUNT(*) / SUM(COUNT(*)) OVER() as percentage_of_total
FROM iceberg.bronze.orders_raw
WHERE total_items > 1
GROUP BY 1
UNION ALL
SELECT
    'single_item' as requirement,
    COUNT(*) as order_count,
    100.0 * COUNT(*) / SUM(COUNT(*)) OVER() as percentage_of_total
FROM iceberg.bronze.orders_raw
WHERE total_items = 1
GROUP BY 1
"""
        dataset_id = create_virtual_dataset(
            client, database_id,
            "vw_orders_by_requirements",
            sql,
            "gold"
        )

    if not dataset_id:
        print(f"{YELLOW}Skipping chart - dataset not available{NC}")
        return None

    chart_id = create_chart(
        client,
        "Orders by Special Requirements",
        "dist_bar",
        dataset_id,
        {
            "viz_type": "dist_bar",
            "metrics": [{"label": "Order Count", "expressionType": "SQL", "sqlExpression": "SUM(order_count)"}],
            "groupby": ["requirement"],
            "columns": [],
            "row_limit": 10,
            "order_desc": True,
            "show_legend": True,
            "y_axis_format": "SMART_NUMBER",
            "color_scheme": "supersetColors"
        },
        "Bar chart showing distribution of orders by special requirements"
    )

    return chart_id


def setup_order_flow_dashboard(client, database_id):
    """Setup the Order Flow Tracker dashboard"""
    print(f"\n{BLUE}═══ Setting up Order Flow Tracker Dashboard ═══{NC}")

    # Try physical table first
    dataset_id = create_dataset(
        client, database_id,
        "gold", "order_flow_summary",
        "Complete order flow with all stage timestamps and durations"
    )

    if not dataset_id:
        # Create virtual dataset from orders and related data
        sql = """
SELECT
    o.order_id,
    o.workflow_id,
    o.customer_id,
    o.priority,
    o.status as current_status,
    CASE
        WHEN s.shipped_at IS NOT NULL THEN 'shipped'
        WHEN p.completed_at IS NOT NULL THEN 'packed'
        WHEN pk.completed_at IS NOT NULL THEN 'picked'
        WHEN w.wave_id IS NOT NULL THEN 'wave_assigned'
        ELSE 'received'
    END as current_stage,
    o.created_at as order_received_at,
    w.assigned_at as wave_assigned_at,
    w.wave_id,
    pk.started_at as picking_started_at,
    pk.completed_at as picking_completed_at,
    pk.task_id as pick_task_id,
    pk.picker_id,
    p.started_at as packing_started_at,
    p.completed_at as packing_completed_at,
    p.task_id as pack_task_id,
    s.shipped_at,
    s.tracking_number,
    s.carrier,
    CAST(DATE_DIFF('minute', pk.started_at, pk.completed_at) AS DOUBLE) as picking_duration_min,
    CAST(DATE_DIFF('minute', p.started_at, p.completed_at) AS DOUBLE) as packing_duration_min,
    CAST(DATE_DIFF('minute', o.created_at, COALESCE(s.shipped_at, CURRENT_TIMESTAMP)) AS DOUBLE) as total_fulfillment_duration_min
FROM iceberg.bronze.orders_raw o
LEFT JOIN iceberg.bronze.waves_raw w ON o.wave_id = w.wave_id
LEFT JOIN iceberg.bronze.pick_tasks_raw pk ON o.order_id = pk.order_id
LEFT JOIN iceberg.bronze.pack_tasks_raw p ON o.order_id = p.order_id
LEFT JOIN iceberg.bronze.shipments_raw s ON o.order_id = s.order_id
"""
        dataset_id = create_virtual_dataset(
            client, database_id,
            "vw_order_flow_summary",
            sql,
            "gold"
        )

    if not dataset_id:
        print(f"{YELLOW}Skipping Order Flow dashboard - dataset not available{NC}")
        return None

    charts = []

    # Chart 1: Order Summary Table
    chart1 = create_chart(
        client,
        "Order Flow - Summary",
        "table",
        dataset_id,
        {
            "viz_type": "table",
            "query_mode": "raw",
            "all_columns": ["order_id", "customer_id", "priority", "current_status", "current_stage", "tracking_number"],
            "row_limit": 100
        },
        "Order summary information"
    )
    charts.append(chart1)

    # Chart 2: Stage Durations
    chart2 = create_chart(
        client,
        "Order Flow - Stage Durations",
        "dist_bar",
        dataset_id,
        {
            "viz_type": "dist_bar",
            "metrics": [
                {"label": "Avg Picking", "expressionType": "SQL", "sqlExpression": "AVG(picking_duration_min)"},
                {"label": "Avg Packing", "expressionType": "SQL", "sqlExpression": "AVG(packing_duration_min)"}
            ],
            "groupby": ["priority"],
            "row_limit": 100,
            "show_legend": True,
            "y_axis_format": "SMART_NUMBER"
        },
        "Average time spent in each stage by priority"
    )
    charts.append(chart2)

    # Chart 3: Timeline Details
    chart3 = create_chart(
        client,
        "Order Flow - Timeline",
        "table",
        dataset_id,
        {
            "viz_type": "table",
            "query_mode": "raw",
            "all_columns": [
                "order_id", "order_received_at", "picking_started_at",
                "picking_completed_at", "packing_started_at", "packing_completed_at",
                "shipped_at", "total_fulfillment_duration_min"
            ],
            "row_limit": 100
        },
        "Order flow timeline with timestamps"
    )
    charts.append(chart3)

    # Create dashboard
    dashboard_id = create_dashboard(
        client,
        "Order Flow Tracker",
        "order-flow-tracker",
        charts,
        "Dashboard for tracking order flow through fulfillment stages"
    )

    return dashboard_id


def main():
    print_banner()

    # Create client with session handling
    client = SupersetClient(SUPERSET_URL, USERNAME, PASSWORD)

    # Create/get database connection
    database_id = create_database_connection(client)

    if not database_id:
        print(f"\n{RED}Failed to setup database connection. Exiting.{NC}")
        sys.exit(1)

    # Setup dashboards
    requirements_chart = setup_orders_requirements_chart(client, database_id)
    order_flow_dash = setup_order_flow_dashboard(client, database_id)

    # Summary
    print(f"\n{BLUE}═══════════════════════════════════════════════════════════{NC}")
    print(f"{GREEN}Setup Complete!{NC}")
    print(f"{BLUE}═══════════════════════════════════════════════════════════{NC}")
    print(f"\nAccess Superset at: {YELLOW}http://localhost:8088{NC}")
    print(f"Login: admin / admin")
    print(f"\nResources created:")
    if requirements_chart:
        print(f"  {GREEN}✓{NC} Orders by Special Requirements (chart)")
    if order_flow_dash:
        print(f"  {GREEN}✓{NC} Order Flow Tracker (dashboard)")
    print(f"\n{YELLOW}Note:{NC} Add native filters for order_id lookups in the Order Flow Tracker.")


if __name__ == "__main__":
    main()
