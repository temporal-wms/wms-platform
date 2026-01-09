#!/usr/bin/env python3
"""
WMS Platform - Superset Dashboard Setup Script
Creates datasets, charts, and dashboards for:
1. Orders by Requirements Bar Chart
2. Order Flow Tracker Dashboard
3. Tote Lookup Dashboard
4. Route Performance by Zone Dashboard
5. Route Optimization Analysis Dashboard
6. Labor Performance Dashboard
7. Inventory Analytics Dashboard
8. Wave Performance Dashboard
9. Receiving Operations Dashboard
10. Operations KPI Dashboard
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

        # First, get the login page to establish session and get CSRF cookie
        login_page = self.session.get(f"{self.base_url}/login/")

        # Extract CSRF token from cookies (set by Superset automatically)
        self.csrf_token = self.session.cookies.get('csrf_access_token')
        if not self.csrf_token:
            # Try alternative cookie names
            self.csrf_token = self.session.cookies.get('_csrf') or self.session.cookies.get('session')

        if self.csrf_token:
            print(f"  Got CSRF token from cookie")

        # Get JWT token with CSRF
        response = self.session.post(
            f"{self.base_url}/api/v1/security/login",
            json={
                "username": username,
                "password": password,
                "provider": "db",
                "refresh": True
            },
            headers={
                "Content-Type": "application/json",
                "X-CSRFToken": self.csrf_token or ""
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

        # Get fresh CSRF token from cookies after login
        self.csrf_token = self.session.cookies.get('csrf_access_token')
        if self.csrf_token:
            self.session.headers.update({"X-CSRFToken": self.csrf_token})

        print(f"{GREEN}✓ Authenticated successfully{NC}")

    def get(self, endpoint, params=None):
        return self.session.get(f"{self.base_url}{endpoint}", params=params)

    def _refresh_csrf(self):
        """Refresh CSRF token from the security endpoint"""
        response = self.session.get(f"{self.base_url}/api/v1/security/csrf_token/")
        if response.status_code == 200:
            self.csrf_token = response.json().get("result")
            self.session.headers.update({"X-CSRFToken": self.csrf_token})
        # Also check cookies
        csrf = self.session.cookies.get('csrf_access_token')
        if csrf and not self.csrf_token:
            self.csrf_token = csrf
            self.session.headers.update({"X-CSRFToken": self.csrf_token})

    def post(self, endpoint, data):
        # Refresh CSRF token before POST
        self._refresh_csrf()
        return self.session.post(f"{self.base_url}{endpoint}", json=data)

    def put(self, endpoint, data):
        self._refresh_csrf()
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


def setup_tote_lookup_dashboard(client, database_id):
    """Setup the Tote Lookup dashboard for consolidation"""
    print(f"\n{BLUE}═══ Setting up Tote Lookup Dashboard ═══{NC}")

    # Create virtual dataset for tote lookup using MongoDB consolidation_db
    # Note: MongoDB column names are lowercase without underscores
    sql = """
SELECT
    consolidationid,
    orderid,
    sourcetotes,
    status,
    station,
    totalexpected,
    totalconsolidated,
    createdat
FROM mongodb.consolidation_db.consolidations
ORDER BY createdat DESC
"""
    dataset_id = create_virtual_dataset(
        client, database_id,
        "vw_orders_by_tote",
        sql,
        "consolidation_db"
    )

    if not dataset_id:
        print(f"{YELLOW}Skipping Tote Lookup dashboard - dataset not available{NC}")
        return None

    charts = []

    # Chart 1: Orders Table
    chart1 = create_chart(
        client,
        "Tote Lookup - Orders",
        "table",
        dataset_id,
        {
            "viz_type": "table",
            "query_mode": "raw",
            "all_columns": ["consolidationid", "orderid", "sourcetotes", "status", "station", "totalexpected", "totalconsolidated", "createdat"],
            "row_limit": 100
        },
        "Orders in consolidation with tote information"
    )
    charts.append(chart1)

    # Create dashboard
    dashboard_id = create_dashboard(
        client,
        "Tote Lookup",
        "tote-lookup",
        charts,
        "Dashboard for looking up orders by tote ID"
    )

    return dashboard_id


def setup_route_performance_dashboard(client, database_id):
    """Setup the Route Performance by Zone dashboard"""
    print(f"\n{BLUE}═══ Setting up Route Performance Dashboard ═══{NC}")

    # Try physical table first
    dataset_id = create_dataset(
        client, database_id,
        "gold", "route_performance_by_zone_daily",
        "Daily route performance metrics by warehouse zone"
    )

    if not dataset_id:
        # Create virtual dataset using MongoDB routing_db
        # Note: MongoDB column names are lowercase without underscores
        sql = """
SELECT
    CAST(createdat AS DATE) AS route_date,
    routeid,
    orderid,
    zone,
    strategy,
    status,
    estimateddistance,
    totalitems,
    createdat
FROM mongodb.routing_db.routes
ORDER BY createdat DESC
"""
        dataset_id = create_virtual_dataset(
            client, database_id,
            "vw_route_performance_by_zone",
            sql,
            "routing_db"
        )

    if not dataset_id:
        print(f"{YELLOW}Skipping Route Performance dashboard - dataset not available{NC}")
        return None

    charts = []

    # Chart 1: Routes by Zone bar chart
    chart1 = create_chart(
        client,
        "Route Performance - Routes by Zone",
        "dist_bar",
        dataset_id,
        {
            "viz_type": "dist_bar",
            "metrics": [{"label": "Total Routes", "expressionType": "SQL", "sqlExpression": "COUNT(*)"}],
            "groupby": ["zone"],
            "row_limit": 20,
            "order_desc": True,
            "show_legend": True,
            "y_axis_format": "SMART_NUMBER",
            "color_scheme": "supersetColors"
        },
        "Number of routes per zone"
    )
    charts.append(chart1)

    # Chart 2: Distance Distribution
    chart2 = create_chart(
        client,
        "Route Performance - Avg Distance",
        "big_number_total",
        dataset_id,
        {
            "viz_type": "big_number_total",
            "metric": {"label": "Avg Distance", "expressionType": "SQL", "sqlExpression": "AVG(estimateddistance)"},
            "y_axis_format": ",.0f"
        },
        "Average estimated distance (m)"
    )
    charts.append(chart2)

    # Chart 3: Details Table
    chart3 = create_chart(
        client,
        "Route Performance - Details",
        "table",
        dataset_id,
        {
            "viz_type": "table",
            "query_mode": "raw",
            "all_columns": ["route_date", "routeid", "orderid", "zone", "strategy", "status", "estimateddistance", "totalitems"],
            "row_limit": 100
        },
        "Route performance details"
    )
    charts.append(chart3)

    # Create dashboard
    dashboard_id = create_dashboard(
        client,
        "Route Performance by Zone",
        "route-performance",
        charts,
        "Dashboard for analyzing picking route efficiency across warehouse zones"
    )

    return dashboard_id


def setup_route_optimization_dashboard(client, database_id):
    """Setup the Route Optimization Analysis dashboard"""
    print(f"\n{BLUE}═══ Setting up Route Optimization Dashboard ═══{NC}")

    # Try physical table first
    dataset_id = create_dataset(
        client, database_id,
        "gold", "route_optimization_candidates",
        "Routes with optimization scores and flags"
    )

    if not dataset_id:
        # Create virtual dataset using MongoDB routing_db
        # Note: MongoDB column names are lowercase without underscores
        sql = """
SELECT
    routeid,
    orderid,
    CAST(createdat AS DATE) AS route_date,
    zone,
    strategy,
    status,
    estimateddistance,
    totalitems,
    CASE WHEN estimateddistance > 500 THEN 2 ELSE 0 END +
    CASE WHEN totalitems > 20 THEN 2 ELSE 0 END AS optimization_score,
    createdat
FROM mongodb.routing_db.routes
ORDER BY createdat DESC
"""
        dataset_id = create_virtual_dataset(
            client, database_id,
            "vw_route_optimization_candidates",
            sql,
            "routing_db"
        )

    if not dataset_id:
        print(f"{YELLOW}Skipping Route Optimization dashboard - dataset not available{NC}")
        return None

    charts = []

    # Chart 1: Optimization Score Distribution
    chart1 = create_chart(
        client,
        "Route Optimization - Score Distribution",
        "dist_bar",
        dataset_id,
        {
            "viz_type": "dist_bar",
            "metrics": [{"label": "Route Count", "expressionType": "SQL", "sqlExpression": "COUNT(*)"}],
            "groupby": ["optimization_score"],
            "row_limit": 20,
            "order_desc": False,
            "show_legend": True,
            "y_axis_format": "SMART_NUMBER",
            "color_scheme": "supersetColors"
        },
        "Distribution of routes by optimization score"
    )
    charts.append(chart1)

    # Chart 2: Routes Needing Attention Table
    chart2 = create_chart(
        client,
        "Route Optimization - Routes Needing Attention",
        "table",
        dataset_id,
        {
            "viz_type": "table",
            "query_mode": "raw",
            "all_columns": ["routeid", "orderid", "route_date", "zone", "strategy", "status", "estimateddistance", "totalitems", "optimization_score"],
            "row_limit": 50,
            "order_by_cols": [["optimization_score", False]]
        },
        "Routes with highest optimization scores"
    )
    charts.append(chart2)

    # Create dashboard
    dashboard_id = create_dashboard(
        client,
        "Route Optimization Analysis",
        "route-optimization",
        charts,
        "Dashboard for identifying routes that need optimization"
    )

    return dashboard_id


# ============================================
# NEW DATA PRODUCT DASHBOARDS
# ============================================

def setup_labor_performance_dashboard(client, database_id):
    """Setup the Labor Performance dashboard"""
    print(f"\n{BLUE}═══ Setting up Labor Performance Dashboard ═══{NC}")

    dataset_id = create_dataset(
        client, database_id,
        "gold", "labor_productivity_daily",
        "Daily labor productivity metrics by worker, zone, and shift"
    )

    if not dataset_id:
        print(f"{YELLOW}Skipping Labor Performance dashboard - dataset not available{NC}")
        return None

    charts = []

    # Chart 1: Total Workers KPI
    chart1 = create_chart(
        client,
        "Labor - Total Workers",
        "big_number_total",
        dataset_id,
        {
            "viz_type": "big_number_total",
            "metric": {"label": "Workers", "expressionType": "SQL", "sqlExpression": "COUNT(DISTINCT worker_id)"},
            "subheader": "Active Workers"
        },
        "Total active workers"
    )
    charts.append(chart1)

    # Chart 2: Avg Items Per Hour
    chart2 = create_chart(
        client,
        "Labor - Avg Items/Hour",
        "big_number_total",
        dataset_id,
        {
            "viz_type": "big_number_total",
            "metric": {"label": "Items/Hour", "expressionType": "SQL", "sqlExpression": "AVG(items_per_hour)"},
            "y_axis_format": ",.1f",
            "subheader": "Items Per Hour"
        },
        "Average items processed per hour"
    )
    charts.append(chart2)

    # Chart 3: Productivity by Zone
    chart3 = create_chart(
        client,
        "Labor - Productivity by Zone",
        "dist_bar",
        dataset_id,
        {
            "viz_type": "dist_bar",
            "metrics": [{"label": "Items/Hour", "expressionType": "SQL", "sqlExpression": "AVG(items_per_hour)"}],
            "groupby": ["zone"],
            "row_limit": 10,
            "order_desc": True,
            "show_legend": True,
            "y_axis_format": ",.1f",
            "color_scheme": "supersetColors"
        },
        "Average productivity by zone"
    )
    charts.append(chart3)

    # Chart 4: Worker Performance Table
    chart4 = create_chart(
        client,
        "Labor - Worker Performance",
        "table",
        dataset_id,
        {
            "viz_type": "table",
            "query_mode": "raw",
            "all_columns": ["date", "worker_id", "worker_name", "zone", "shift_type", "tasks_completed", "items_processed", "items_per_hour", "accuracy_rate"],
            "row_limit": 100
        },
        "Worker performance details"
    )
    charts.append(chart4)

    dashboard_id = create_dashboard(
        client,
        "Labor Performance",
        "labor-performance",
        charts,
        "Dashboard for monitoring workforce productivity and utilization"
    )

    return dashboard_id


def setup_inventory_analytics_dashboard(client, database_id):
    """Setup the Inventory Analytics dashboard"""
    print(f"\n{BLUE}═══ Setting up Inventory Analytics Dashboard ═══{NC}")

    dataset_id = create_dataset(
        client, database_id,
        "silver", "inventory_current",
        "Current inventory levels and stock status"
    )

    if not dataset_id:
        print(f"{YELLOW}Skipping Inventory Analytics dashboard - dataset not available{NC}")
        return None

    charts = []

    # Chart 1: Total SKUs
    chart1 = create_chart(
        client,
        "Inventory - Total SKUs",
        "big_number_total",
        dataset_id,
        {
            "viz_type": "big_number_total",
            "metric": {"label": "SKUs", "expressionType": "SQL", "sqlExpression": "COUNT(DISTINCT sku)"},
            "subheader": "SKUs"
        },
        "Total unique SKUs"
    )
    charts.append(chart1)

    # Chart 2: Total Units
    chart2 = create_chart(
        client,
        "Inventory - Total Units",
        "big_number_total",
        dataset_id,
        {
            "viz_type": "big_number_total",
            "metric": {"label": "Units", "expressionType": "SQL", "sqlExpression": "SUM(total_quantity)"},
            "y_axis_format": ",",
            "subheader": "Units"
        },
        "Total units in inventory"
    )
    charts.append(chart2)

    # Chart 3: Low Stock Alerts
    chart3 = create_chart(
        client,
        "Inventory - Low Stock Alerts",
        "big_number_total",
        dataset_id,
        {
            "viz_type": "big_number_total",
            "metric": {"label": "Low Stock", "expressionType": "SQL", "sqlExpression": "SUM(CASE WHEN available_quantity <= reorder_point THEN 1 ELSE 0 END)"},
            "subheader": "Alerts"
        },
        "SKUs below reorder point"
    )
    charts.append(chart3)

    # Chart 4: Inventory Table
    chart4 = create_chart(
        client,
        "Inventory - Reorder Alerts",
        "table",
        dataset_id,
        {
            "viz_type": "table",
            "query_mode": "raw",
            "all_columns": ["sku", "product_name", "total_quantity", "available_quantity", "reserved_quantity", "reorder_point"],
            "row_limit": 100
        },
        "SKUs requiring reorder"
    )
    charts.append(chart4)

    dashboard_id = create_dashboard(
        client,
        "Inventory Analytics",
        "inventory-analytics",
        charts,
        "Dashboard for stock visibility and inventory health monitoring"
    )

    return dashboard_id


def setup_wave_performance_dashboard(client, database_id):
    """Setup the Wave Performance dashboard"""
    print(f"\n{BLUE}═══ Setting up Wave Performance Dashboard ═══{NC}")

    dataset_id = create_dataset(
        client, database_id,
        "gold", "wave_performance_daily",
        "Daily wave execution metrics by wave type"
    )

    if not dataset_id:
        print(f"{YELLOW}Skipping Wave Performance dashboard - dataset not available{NC}")
        return None

    charts = []

    # Chart 1: Total Waves
    chart1 = create_chart(
        client,
        "Wave - Total Waves",
        "big_number_total",
        dataset_id,
        {
            "viz_type": "big_number_total",
            "metric": {"label": "Waves", "expressionType": "SQL", "sqlExpression": "SUM(waves_created)"},
            "subheader": "Waves Created"
        },
        "Total waves created"
    )
    charts.append(chart1)

    # Chart 2: Completion Rate
    chart2 = create_chart(
        client,
        "Wave - Completion Rate",
        "big_number_total",
        dataset_id,
        {
            "viz_type": "big_number_total",
            "metric": {"label": "Rate", "expressionType": "SQL", "sqlExpression": "100.0 * SUM(waves_completed) / NULLIF(SUM(waves_created), 0)"},
            "y_axis_format": ",.1f",
            "subheader": "% Completion"
        },
        "Wave completion rate"
    )
    charts.append(chart2)

    # Chart 3: Waves by Type
    chart3 = create_chart(
        client,
        "Wave - By Type",
        "dist_bar",
        dataset_id,
        {
            "viz_type": "dist_bar",
            "metrics": [{"label": "Waves", "expressionType": "SQL", "sqlExpression": "SUM(waves_created)"}],
            "groupby": ["wave_type"],
            "row_limit": 10,
            "order_desc": True,
            "show_legend": True,
            "color_scheme": "supersetColors"
        },
        "Waves by type"
    )
    charts.append(chart3)

    # Chart 4: Wave Details Table
    chart4 = create_chart(
        client,
        "Wave - Performance Details",
        "table",
        dataset_id,
        {
            "viz_type": "table",
            "query_mode": "raw",
            "all_columns": ["date", "wave_type", "waves_created", "waves_completed", "waves_cancelled", "avg_orders_per_wave", "avg_completion_time_hours"],
            "row_limit": 100
        },
        "Wave performance details"
    )
    charts.append(chart4)

    dashboard_id = create_dashboard(
        client,
        "Wave Performance",
        "wave-performance",
        charts,
        "Dashboard for monitoring wave creation, completion rates, and cycle times"
    )

    return dashboard_id


def setup_receiving_dashboard(client, database_id):
    """Setup the Receiving Operations dashboard"""
    print(f"\n{BLUE}═══ Setting up Receiving Dashboard ═══{NC}")

    dataset_id = create_dataset(
        client, database_id,
        "gold", "receiving_metrics_daily",
        "Daily receiving metrics by dock door"
    )

    if not dataset_id:
        print(f"{YELLOW}Skipping Receiving dashboard - dataset not available{NC}")
        return None

    charts = []

    # Chart 1: Total Receipts
    chart1 = create_chart(
        client,
        "Receiving - Total Receipts",
        "big_number_total",
        dataset_id,
        {
            "viz_type": "big_number_total",
            "metric": {"label": "Receipts", "expressionType": "SQL", "sqlExpression": "SUM(total_receipts)"},
            "subheader": "Receipts"
        },
        "Total receipts processed"
    )
    charts.append(chart1)

    # Chart 2: Dock-to-Stock Time
    chart2 = create_chart(
        client,
        "Receiving - Dock-to-Stock",
        "big_number_total",
        dataset_id,
        {
            "viz_type": "big_number_total",
            "metric": {"label": "Minutes", "expressionType": "SQL", "sqlExpression": "AVG(avg_dock_to_stock_minutes)"},
            "y_axis_format": ",.0f",
            "subheader": "Avg Minutes"
        },
        "Average dock-to-stock time"
    )
    charts.append(chart2)

    # Chart 3: Units by Dock Door
    chart3 = create_chart(
        client,
        "Receiving - Units by Dock",
        "dist_bar",
        dataset_id,
        {
            "viz_type": "dist_bar",
            "metrics": [{"label": "Units", "expressionType": "SQL", "sqlExpression": "SUM(total_units_received)"}],
            "groupby": ["dock_door"],
            "row_limit": 10,
            "order_desc": True,
            "show_legend": True,
            "color_scheme": "supersetColors"
        },
        "Units received by dock door"
    )
    charts.append(chart3)

    # Chart 4: Receiving Details Table
    chart4 = create_chart(
        client,
        "Receiving - Details",
        "table",
        dataset_id,
        {
            "viz_type": "table",
            "query_mode": "raw",
            "all_columns": ["date", "dock_door", "total_receipts", "completed_receipts", "total_units_received", "receiving_accuracy", "avg_dock_to_stock_minutes", "on_time_rate"],
            "row_limit": 100
        },
        "Receiving performance details"
    )
    charts.append(chart4)

    dashboard_id = create_dashboard(
        client,
        "Receiving Operations",
        "receiving",
        charts,
        "Dashboard for monitoring inbound operations and vendor performance"
    )

    return dashboard_id


def setup_operations_kpi_dashboard(client, database_id):
    """Setup the Operations KPI executive dashboard"""
    print(f"\n{BLUE}═══ Setting up Operations KPI Dashboard ═══{NC}")

    dataset_id = create_dataset(
        client, database_id,
        "gold", "operations_summary_daily",
        "Daily operations summary with key metrics across all domains"
    )

    if not dataset_id:
        print(f"{YELLOW}Skipping Operations KPI dashboard - dataset not available{NC}")
        return None

    charts = []

    # Chart 1: Orders Received
    chart1 = create_chart(
        client,
        "KPI - Orders Today",
        "big_number_total",
        dataset_id,
        {
            "viz_type": "big_number_total",
            "metric": {"label": "Orders", "expressionType": "SQL", "sqlExpression": "SUM(orders_received)"},
            "subheader": "Orders"
        },
        "Total orders received"
    )
    charts.append(chart1)

    # Chart 2: Fulfillment Rate
    chart2 = create_chart(
        client,
        "KPI - Fulfillment Rate",
        "big_number_total",
        dataset_id,
        {
            "viz_type": "big_number_total",
            "metric": {"label": "Rate", "expressionType": "SQL", "sqlExpression": "100.0 * SUM(orders_completed) / NULLIF(SUM(orders_received), 0)"},
            "y_axis_format": ",.1f",
            "subheader": "% Fulfillment"
        },
        "Order fulfillment rate"
    )
    charts.append(chart2)

    # Chart 3: Items Picked
    chart3 = create_chart(
        client,
        "KPI - Items Picked",
        "big_number_total",
        dataset_id,
        {
            "viz_type": "big_number_total",
            "metric": {"label": "Items", "expressionType": "SQL", "sqlExpression": "SUM(items_picked)"},
            "y_axis_format": ",",
            "subheader": "Items"
        },
        "Total items picked"
    )
    charts.append(chart3)

    # Chart 4: Shipments Sent
    chart4 = create_chart(
        client,
        "KPI - Shipments",
        "big_number_total",
        dataset_id,
        {
            "viz_type": "big_number_total",
            "metric": {"label": "Shipments", "expressionType": "SQL", "sqlExpression": "SUM(shipments_sent)"},
            "subheader": "Shipments"
        },
        "Total shipments sent"
    )
    charts.append(chart4)

    # Chart 5: Active Workers
    chart5 = create_chart(
        client,
        "KPI - Active Workers",
        "big_number_total",
        dataset_id,
        {
            "viz_type": "big_number_total",
            "metric": {"label": "Workers", "expressionType": "SQL", "sqlExpression": "SUM(active_workers)"},
            "subheader": "Workers"
        },
        "Active workers"
    )
    charts.append(chart5)

    # Chart 6: Operations Summary Table
    chart6 = create_chart(
        client,
        "KPI - Operations Summary",
        "table",
        dataset_id,
        {
            "viz_type": "table",
            "query_mode": "raw",
            "all_columns": ["date", "orders_received", "orders_completed", "order_completion_rate", "items_picked", "packages_packed", "shipments_sent", "receipts_completed", "returns_processed", "active_workers"],
            "row_limit": 30
        },
        "Daily operations summary"
    )
    charts.append(chart6)

    dashboard_id = create_dashboard(
        client,
        "Operations KPI",
        "operations-kpi",
        charts,
        "Executive-level overview of warehouse operations with key performance indicators"
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
    tote_lookup_dash = setup_tote_lookup_dashboard(client, database_id)
    route_performance_dash = setup_route_performance_dashboard(client, database_id)
    route_optimization_dash = setup_route_optimization_dashboard(client, database_id)

    # Setup new data product dashboards
    labor_performance_dash = setup_labor_performance_dashboard(client, database_id)
    inventory_analytics_dash = setup_inventory_analytics_dashboard(client, database_id)
    wave_performance_dash = setup_wave_performance_dashboard(client, database_id)
    receiving_dash = setup_receiving_dashboard(client, database_id)
    operations_kpi_dash = setup_operations_kpi_dashboard(client, database_id)

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
    if tote_lookup_dash:
        print(f"  {GREEN}✓{NC} Tote Lookup (dashboard)")
    if route_performance_dash:
        print(f"  {GREEN}✓{NC} Route Performance by Zone (dashboard)")
    if route_optimization_dash:
        print(f"  {GREEN}✓{NC} Route Optimization Analysis (dashboard)")
    if labor_performance_dash:
        print(f"  {GREEN}✓{NC} Labor Performance (dashboard)")
    if inventory_analytics_dash:
        print(f"  {GREEN}✓{NC} Inventory Analytics (dashboard)")
    if wave_performance_dash:
        print(f"  {GREEN}✓{NC} Wave Performance (dashboard)")
    if receiving_dash:
        print(f"  {GREEN}✓{NC} Receiving Operations (dashboard)")
    if operations_kpi_dash:
        print(f"  {GREEN}✓{NC} Operations KPI (dashboard)")
    print(f"\n{YELLOW}Note:{NC} Add native filters for order_id/tote_id lookups in the dashboards.")


if __name__ == "__main__":
    main()
