// Command gen writes the Kestrel Group seed: source-shaped exports for six tools, the answer key
// (seed/truth.yaml) and the recorded model judgements (seed/judgements.jsonl).
//
// The Kestrel Group is fictional. The generator starts from one true model of the company and renders
// it into what each tool would export, with the mess real exports have: different names in every
// system, missing keys, a stale EA repository, an application nobody in group IT knows about.
// Every planted problem is listed in planted below and must be found by the running system.
//
// Run with `make seed-gen`. The output is committed; the generator is deterministic.
package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	yaml "go.yaml.in/yaml/v3"
)

// Ref is the reference time of the seed: "now" for every export and for scoring.
var Ref = time.Date(2026, 11, 12, 3, 0, 0, 0, time.UTC)

func daysAgo(d int) time.Time { return Ref.Add(-time.Duration(d) * 24 * time.Hour) }
func ts(t time.Time) string   { return t.Format(time.RFC3339) }

type MU struct{ Slug, EAID, Name string }

type Team struct {
	Slug, Name, MU, EAID string
}

type Cap struct {
	Slug, EAID, Name, Parent string
}

type Repo struct {
	Name       string
	Key        bool   // carries the EA application id in catalog-info.yaml
	Committers int    // distinct committers in twelve months
	PushedDays int    // days since last push
	Readme     string // first lines of the README
	Team       string // owning team on the git host (defaults to the app owner)
}

type Workload struct {
	NS, Name string
	Label    bool   // carries app.kestrel.example/application-id
	Type     string // Deployment or CronJob
	Repo     string // source repository label, when set
	NoSource bool   // no source repository label at all
}

type Endpoint struct {
	Host, Path, Exposure, WAF string // WAF "" means no policy at all
	Upstream                  string // workload name in the app's first namespace
}

type App struct {
	Short, Name, EAID, Team string
	MUs, Caps               []string
	Lifecycle               string
	Repos                   []Repo
	Workloads               []Workload
	Endpoints               []Endpoint
	Deps                    []string
	Successor               string
	ITSMName                string // business service name in IT service management ("" means none)
	ITSMNoRef               bool   // the business service has no EA reference
	EAOwnerBlank            bool   // the EA record has no owner
	EAStaleDays             int    // days since the EA record was touched
	InEA                    bool
	Incidents               int
	Desc                    string
}

var mus = []MU{
	{"north", "MU-N", "North"},
	{"central", "MU-C", "Central"},
	{"south", "MU-S", "South"},
}

var teams = []Team{
	{"claims-platform", "Claims Platform", "central", "ORG-11"},
	{"policy-core", "Policy Core", "central", "ORG-12"},
	{"payments", "Payments", "central", "ORG-13"},
	{"customer-digital", "Customer Digital", "north", "ORG-14"},
	{"broker-channel", "Broker Channel", "north", "ORG-15"},
	{"finance-systems", "Finance Systems", "central", "ORG-16"},
	{"risk-analytics", "Risk Analytics", "central", "ORG-17"},
	{"data-platform", "Data Platform", "central", "ORG-18"},
	{"integration-services", "Integration Services", "central", "ORG-19"},
	{"identity-and-access", "Identity and Access", "central", "ORG-20"},
	{"south-digital", "South Digital", "south", "ORG-21"},
	{"platform-engineering", "Platform Engineering", "central", "ORG-22"},
}

var caps = []Cap{
	{"policy-management", "CAP-01", "Policy Management", ""},
	{"product-configuration", "CAP-01.1", "Product Configuration", "policy-management"},
	{"policy-servicing", "CAP-01.2", "Policy Servicing", "policy-management"},
	{"underwriting", "CAP-01.3", "Underwriting", "policy-management"},
	{"claims-management", "CAP-02", "Claims Management", ""},
	{"claims-intake", "CAP-02.1", "Claims Intake", "claims-management"},
	{"claims-settlement", "CAP-02.2", "Claims Settlement", "claims-management"},
	{"payments", "CAP-03", "Payments", ""},
	{"payment-processing", "CAP-03.1", "Payment Processing", "payments"},
	{"collections", "CAP-03.2", "Collections", "payments"},
	{"customer-engagement", "CAP-04", "Customer Engagement", ""},
	{"customer-self-service", "CAP-04.1", "Customer Self Service", "customer-engagement"},
	{"customer-communication", "CAP-04.2", "Customer Communication", "customer-engagement"},
	{"customer-data", "CAP-04.3", "Customer Data", "customer-engagement"},
	{"document-management", "CAP-04.4", "Document Management", "customer-engagement"},
	{"distribution", "CAP-05", "Distribution", ""},
	{"broker-management", "CAP-05.1", "Broker Management", "distribution"},
	{"commission", "CAP-05.2", "Commission", "distribution"},
	{"finance", "CAP-06", "Finance", ""},
	{"general-ledger", "CAP-06.1", "General Ledger", "finance"},
	{"financial-reporting", "CAP-06.2", "Financial Reporting", "finance"},
	{"treasury", "CAP-06.3", "Treasury", "finance"},
	{"risk-and-compliance", "CAP-07", "Risk and Compliance", ""},
	{"fraud-prevention", "CAP-07.4", "Fraud Prevention", "risk-and-compliance"},
	{"risk-modelling", "CAP-07.1", "Risk Modelling", "risk-and-compliance"},
	{"regulatory-reporting", "CAP-07.2", "Regulatory Reporting", "risk-and-compliance"},
	{"capital-management", "CAP-07.3", "Capital Management", "risk-and-compliance"},
	{"integration", "CAP-08", "Integration", ""},
	{"messaging", "CAP-08.1", "Messaging", "integration"},
	{"api-management", "CAP-08.2", "API Management", "integration"},
	{"data-and-analytics", "CAP-09", "Data and Analytics", ""},
	{"data-warehouse", "CAP-09.1", "Data Warehouse", "data-and-analytics"},
	{"analytics", "CAP-09.2", "Analytics", "data-and-analytics"},
	{"security", "CAP-10", "Security", ""},
	{"identity-and-access-management", "CAP-10.1", "Identity and Access Management", "security"},
}

func r(name string, key bool, committers, pushed int, readme string) Repo {
	return Repo{Name: name, Key: key, Committers: committers, PushedDays: pushed, Readme: readme}
}

func w(ns, name string, label bool) Workload {
	return Workload{NS: ns, Name: name, Label: label, Type: "Deployment"}
}

// apps is the truth: what Kestrel really runs. InEA marks what the EA repository knows about.
var apps = []App{
	// claims-platform
	{Short: "policy-admin", Name: "Policy Administration", EAID: "APP-0142", Team: "claims-platform",
		MUs: []string{"central", "north"}, Caps: []string{"policy-servicing"}, Lifecycle: "production",
		Repos: []Repo{
			r("pol-adm-svc", true, 7, 2, "Policy administration service: contracts, endorsements, renewals."),
			r("pol-adm-ui", false, 4, 5, "Frontend for policy administration. Talks to pol-adm-svc."),
		},
		Workloads: []Workload{w("claims-prod", "policy-admin", true), {NS: "claims-prod", Name: "policy-admin-ui", Label: true, Type: "Deployment", NoSource: true}},
		Endpoints: []Endpoint{{"api.kestrel.example", "/policies", "internet", "enforced", "policy-admin"}},
		Deps:      []string{"payment-gateway", "customer-master", "document-service"},
		ITSMName:  "PAS", ITSMNoRef: true, EAStaleDays: 40, InEA: true, Incidents: 3,
		Desc: "Policy contracts, endorsements and renewals for retail lines."},
	{Short: "claims-intake", Name: "Claims Intake", EAID: "APP-0150", Team: "claims-platform",
		MUs: []string{"central", "north"}, Caps: []string{"claims-intake"}, Lifecycle: "production",
		Repos: []Repo{
			r("claims-intake", false, 5, 3, "First notice of loss: receives and triages new claims."),
			r("claims-intake-web", false, 3, 9, "Web form for reporting a claim online."),
		},
		Workloads: []Workload{w("claims-prod", "claims-intake", true)},
		Endpoints: []Endpoint{{"claims.kestrel.example", "/", "internet", "enforced", "claims-intake"}},
		Deps:      []string{"policy-admin", "document-service"},
		ITSMName:  "Claims Intake", EAStaleDays: 220, InEA: true, Incidents: 5,
		Desc: "First notice of loss and claim triage."},
	{Short: "claims-settlement", Name: "Claims Settlement", EAID: "APP-0151", Team: "claims-platform",
		MUs: []string{"central", "north"}, Caps: []string{"claims-settlement"}, Lifecycle: "production",
		Repos:     []Repo{r("clm-settle", true, 4, 6, "Claims settlement and reserve calculation.")},
		Workloads: []Workload{w("claims-prod", "claims-settlement", true)},
		Deps:      []string{"payment-gateway", "policy-admin", "general-ledger"},
		ITSMName:  "Claims Settlement", EAStaleDays: 300, InEA: true, Incidents: 1,
		Desc: "Settlement, reserves and payout instructions for claims."},
	// policy-core
	{Short: "product-catalogue", Name: "Product Catalogue", EAID: "APP-0160", Team: "policy-core",
		MUs: []string{"central", "north"}, Caps: []string{"product-configuration"}, Lifecycle: "production",
		Repos:     []Repo{r("product-catalogue", true, 3, 20, "Insurance products, tariffs and coverages.")},
		Workloads: []Workload{w("policy-prod", "product-catalogue", true)},
		ITSMName:  "Product Catalogue", EAStaleDays: 400, InEA: true,
		Desc: "Products, tariffs and coverages."},
	{Short: "rating-engine", Name: "Rating Engine", EAID: "APP-0161", Team: "policy-core",
		MUs: []string{"central"}, Caps: []string{"underwriting"}, Lifecycle: "production",
		Repos:     []Repo{r("rating-engine", true, 1, 12, "Premium calculation. Ask Jonas before changing anything.")},
		Workloads: []Workload{w("policy-prod", "rating-engine", true)},
		Deps:      []string{"product-catalogue"},
		ITSMName:  "Rating Engine", EAStaleDays: 260, InEA: true, Incidents: 2,
		Desc: "Premium calculation for all retail products."},
	{Short: "policy-batch", Name: "Policy Batch", EAID: "APP-0162", Team: "policy-core",
		MUs: []string{"central"}, Lifecycle: "production",
		Repos: []Repo{r("policy-batch", false, 2, 45, "Nightly renewal and indexation jobs.")},
		Workloads: []Workload{{NS: "policy-prod", Name: "policy-batch-nightly", Label: false, Type: "CronJob",
			Repo: ""}},
		Deps:     []string{"policy-admin"},
		ITSMName: "Policy Batch", EAStaleDays: 500, InEA: true,
		Desc: "Nightly renewals and indexation."},
	{Short: "quote-classic", Name: "Quote Engine Classic", EAID: "APP-0098", Team: "policy-core",
		MUs: []string{"central"}, Caps: []string{"underwriting"}, Lifecycle: "retired",
		Repos:     []Repo{r("quote-classic", true, 0, 700, "Legacy quoting. Replaced by rating-engine.")},
		Workloads: []Workload{w("policy-prod", "quote-classic", true)},
		Successor: "rating-engine", EAStaleDays: 90, InEA: true,
		Desc: "Legacy quoting engine."},
	{Short: "uw-workbench", Name: "Underwriting Workbench", EAID: "APP-0163", Team: "policy-core",
		MUs: []string{"north"}, Caps: []string{"underwriting"}, Lifecycle: "production",
		Repos:     []Repo{r("uw-workbench", true, 4, 15, "Underwriter case handling for commercial lines.")},
		Workloads: []Workload{w("policy-prod", "uw-workbench", true)},
		Deps:      []string{"rating-engine", "product-catalogue"},
		ITSMName:  "Underwriting Workbench", EAStaleDays: 330, InEA: true,
		Desc: "Case handling for underwriters."},
	// payments
	{Short: "payment-gateway", Name: "Payment Gateway", EAID: "APP-0170", Team: "payments",
		MUs: []string{"central", "north", "south"}, Caps: []string{"payment-processing"}, Lifecycle: "production",
		Repos:     []Repo{r("payment-gateway", true, 6, 1, "Card, direct debit and bank transfer payments.")},
		Workloads: []Workload{w("payments-prod", "payment-gateway", true)},
		Endpoints: []Endpoint{{"pay.kestrel.example", "/api", "internet", "enforced", "payment-gateway"}},
		Deps:      []string{"fraud-screening"},
		ITSMName:  "Payment Gateway", EAStaleDays: 30, InEA: true, Incidents: 4,
		Desc: "Inbound and outbound payments for the group."},
	{Short: "collections", Name: "Collections", EAID: "APP-0171", Team: "payments",
		MUs: []string{"central", "north"}, Caps: []string{"collections"}, Lifecycle: "production",
		Repos:     []Repo{r("collections-svc", true, 3, 10, "Dunning and collections.")},
		Workloads: []Workload{w("payments-prod", "collections", true)},
		Deps:      []string{"payment-gateway"},
		ITSMName:  "Collections", EAStaleDays: 410, InEA: true,
		Desc: "Dunning and collections."},
	{Short: "fraud-screening", Name: "Fraud Screening", EAID: "APP-0172", Team: "payments",
		MUs: []string{"central"}, Caps: []string{"fraud-prevention"}, Lifecycle: "production",
		Repos:     []Repo{r("fraud-screening", true, 3, 8, "Rules and scores for payment fraud.")},
		Workloads: []Workload{w("payments-prod", "fraud-screening", true)},
		ITSMName:  "Fraud Screening", EAStaleDays: 250, InEA: true,
		Desc: "Payment fraud rules and scores."},
	// customer-digital
	{Short: "customer-portal", Name: "Customer Portal", EAID: "APP-0180", Team: "customer-digital",
		MUs: []string{"north", "central"}, Caps: []string{"customer-self-service"}, Lifecycle: "production",
		Repos: []Repo{
			r("customer-portal-web", true, 8, 1, "The customer portal frontend."),
			r("customer-portal-bff", true, 5, 2, "Backend for frontend of the customer portal."),
		},
		Workloads: []Workload{w("customer-prod", "customer-portal", true), w("customer-prod", "customer-portal-bff", true)},
		Endpoints: []Endpoint{{"www.kestrel.example", "/", "internet", "enforced", "customer-portal"}},
		Deps:      []string{"customer-master", "policy-admin", "identity-hub", "payment-gateway"},
		ITSMName:  "Customer Portal", EAStaleDays: 60, InEA: true, Incidents: 6,
		Desc: "Self service for customers: policies, claims, documents."},
	{Short: "customer-master", Name: "Customer Master", EAID: "APP-0181", Team: "customer-digital",
		MUs: []string{"north", "central"}, Caps: []string{"customer-data"}, Lifecycle: "production",
		Repos:     []Repo{r("customer-master", true, 4, 14, "Golden record for customers and partners.")},
		Workloads: []Workload{w("customer-prod", "customer-master", true)},
		ITSMName:  "Customer Master Data", EAStaleDays: 380, InEA: true, Incidents: 1,
		Desc: "Golden record for customers."},
	{Short: "notify", Name: "Notification Service", EAID: "APP-0182", Team: "customer-digital",
		MUs: []string{"north", "central"}, Caps: []string{"customer-communication"}, Lifecycle: "production",
		Repos:     []Repo{r("notify-svc", false, 2, 30, "E-mail, SMS and push notifications.")},
		Workloads: []Workload{w("customer-prod", "notify", true)},
		ITSMName:  "Notification Service", EAStaleDays: 365, InEA: true,
		Desc: "E-mail, SMS and push notifications."},
	{Short: "kestrel-mobile", Name: "Kestrel Mobile", EAID: "APP-0183", Team: "customer-digital",
		MUs: []string{"south"}, Caps: []string{"customer-communication"}, Lifecycle: "production",
		Repos:    []Repo{r("kestrel-mobile", true, 3, 7, "The mobile app, iOS and Android.")},
		Deps:     []string{"customer-portal"},
		ITSMName: "Kestrel Mobile", EAStaleDays: 190, InEA: true,
		Desc: "Mobile app for customers."},
	// broker-channel
	{Short: "broker-portal", Name: "Broker Portal", EAID: "APP-0190", Team: "broker-channel",
		MUs: []string{"north"}, Caps: []string{"broker-management"}, Lifecycle: "production",
		Repos:     []Repo{r("broker-portal", true, 4, 4, "Portal for independent brokers.")},
		Workloads: []Workload{w("broker-prod", "broker-portal", true)},
		Endpoints: []Endpoint{{"brokers.kestrel.example", "/", "internet", "", "broker-portal"}},
		Deps:      []string{"policy-admin", "commission"},
		ITSMName:  "Broker Portal", EAStaleDays: 280, InEA: true, Incidents: 2,
		Desc: "Portal for independent brokers."},
	{Short: "commission", Name: "Commission Engine", EAID: "APP-0191", Team: "broker-channel",
		MUs: []string{"north"}, Caps: []string{"commission"}, Lifecycle: "production",
		Repos:     []Repo{r("commission-engine", true, 2, 25, "Broker commission calculation and statements.")},
		Workloads: []Workload{w("broker-prod", "commission", true)},
		Deps:      []string{"general-ledger"},
		ITSMName:  "Commission Engine", EAStaleDays: 450, InEA: true,
		Desc: "Broker commissions."},
	{Short: "partner-api", Name: "Partner API", EAID: "APP-0192", Team: "broker-channel",
		MUs: []string{"north"}, Lifecycle: "production",
		Repos:     []Repo{r("partner-api", true, 3, 11, "REST API for comparison sites and partners.")},
		Workloads: []Workload{w("broker-prod", "partner-api", true)},
		Endpoints: []Endpoint{{"partners.kestrel.example", "/v1", "internet", "monitor", "partner-api"}},
		Deps:      []string{"rating-engine"},
		ITSMName:  "Partner API", EAStaleDays: 210, InEA: true,
		Desc: "API for partners and comparison sites."},
	// finance-systems
	{Short: "general-ledger", Name: "General Ledger", EAID: "APP-0200", Team: "finance-systems",
		MUs: []string{"central", "north", "south"}, Caps: []string{"general-ledger"}, Lifecycle: "production",
		Repos:     []Repo{r("gl-core", true, 3, 18, "General ledger integration and postings.")},
		Workloads: []Workload{w("finance-prod", "general-ledger", true)},
		ITSMName:  "General Ledger", EAStaleDays: 500, InEA: true,
		Desc: "Group general ledger."},
	{Short: "fin-reporting", Name: "Financial Reporting", EAID: "APP-0201", Team: "finance-systems",
		MUs: []string{"central"}, Caps: []string{"financial-reporting"}, Lifecycle: "production",
		Repos: []Repo{
			r("fin-reporting", true, 2, 22, "Statutory and management reporting."),
			r("report-templates", false, 2, 60, "Shared report templates for finance reporting."),
		},
		Workloads: []Workload{w("finance-prod", "fin-reporting", true)},
		Deps:      []string{"general-ledger"},
		ITSMName:  "Financial Reporting", EAStaleDays: 420, InEA: true,
		Desc: "Statutory and management reporting."},
	{Short: "payhub", Name: "PayHub", EAID: "APP-0175", Team: "finance-systems",
		MUs: []string{"central"}, Caps: []string{"payment-processing"}, Lifecycle: "production",
		Repos:     []Repo{r("payhub", true, 2, 35, "Outgoing payments for finance: suppliers and refunds.")},
		Workloads: []Workload{w("finance-prod", "payhub", true)},
		Deps:      []string{"general-ledger"},
		ITSMName:  "PayHub", EAStaleDays: 600, InEA: true, Incidents: 1,
		Desc: "Outgoing payments for finance."},
	{Short: "treasury", Name: "Treasury Management", EAID: "APP-0202", Team: "finance-systems",
		MUs: []string{"central"}, Caps: []string{"treasury"}, Lifecycle: "production",
		Repos:     []Repo{r("treasury", true, 2, 40, "Cash positions and liquidity planning.")},
		Workloads: []Workload{w("finance-prod", "treasury", true)},
		ITSMName:  "Treasury Management", EAStaleDays: 520, InEA: true,
		Desc: "Cash and liquidity."},
	// risk-analytics
	{Short: "risk-models", Name: "Risk Models", EAID: "APP-0210", Team: "risk-analytics",
		MUs: []string{"central"}, Caps: []string{"risk-modelling"}, Lifecycle: "production",
		Repos:     []Repo{r("risk-models", true, 5, 6, "Actuarial and risk models.")},
		Workloads: []Workload{w("risk-prod", "risk-models", true)},
		ITSMName:  "Risk Models", EAStaleDays: 230, InEA: true,
		Desc: "Actuarial and risk models."},
	{Short: "reg-reporting", Name: "Regulatory Reporting", EAID: "APP-0211", Team: "risk-analytics",
		MUs: []string{"central", "north", "south"}, Caps: []string{"regulatory-reporting"}, Lifecycle: "production",
		Repos:     []Repo{r("reg-reporting", true, 3, 16, "Supervisory reporting.")},
		Workloads: []Workload{w("risk-prod", "reg-reporting", true)},
		Deps:      []string{"data-warehouse", "fin-reporting"},
		ITSMName:  "Regulatory Reporting", EAStaleDays: 200, InEA: true,
		Desc: "Supervisory reporting."},
	{Short: "solvency-calc", Name: "Solvency Calculator", EAID: "APP-0212", Team: "risk-analytics",
		MUs: []string{"central"}, Caps: []string{"capital-management"}, Lifecycle: "production",
		Repos:        []Repo{r("solvency-calc", true, 2, 50, "Solvency capital calculation.")},
		Workloads:    []Workload{w("risk-prod", "solvency-calc", true)},
		Deps:         []string{"risk-models"},
		EAOwnerBlank: true, EAStaleDays: 700, InEA: true,
		Desc: "Solvency capital calculation."},
	// data-platform
	{Short: "data-warehouse", Name: "Enterprise Data Warehouse", EAID: "APP-0220", Team: "data-platform",
		MUs: []string{"central", "north", "south"}, Caps: []string{"data-warehouse"}, Lifecycle: "production",
		Repos:     []Repo{r("edw-pipelines", true, 6, 3, "Loads and models for the enterprise data warehouse.")},
		Workloads: []Workload{w("data-prod", "edw-loader", true)},
		ITSMName:  "Enterprise Data Warehouse", EAStaleDays: 150, InEA: true, Incidents: 2,
		Desc: "Enterprise data warehouse."},
	{Short: "analytics-workbench", Name: "Analytics Workbench", EAID: "APP-0221", Team: "data-platform",
		MUs: []string{"central"}, Caps: []string{"analytics"}, Lifecycle: "production",
		Repos:     []Repo{r("analytics-workbench", true, 3, 9, "Notebooks and dashboards for analysts.")},
		Workloads: []Workload{w("data-prod", "workbench", true)},
		Deps:      []string{"data-warehouse"},
		ITSMName:  "Analytics Workbench", EAStaleDays: 240, InEA: true,
		Desc: "Notebooks and dashboards for analysts."},
	{Short: "market-insight", Name: "Market Insight Portal", EAID: "APP-0108", Team: "data-platform",
		MUs: []string{"north"}, Caps: []string{"analytics"}, Lifecycle: "production",
		EAStaleDays: 420, InEA: true,
		Desc: "Market and competitor dashboards for sales."},
	// integration-services
	{Short: "esb", Name: "Enterprise Service Bus", EAID: "APP-0230", Team: "integration-services",
		MUs: []string{"central", "north"}, Caps: []string{"messaging"}, Lifecycle: "production",
		Repos:     []Repo{r("esb-config", true, 3, 5, "Flows and configuration of the group service bus.")},
		Workloads: []Workload{w("integration-prod", "esb", true)},
		ITSMName:  "Enterprise Service Bus", EAStaleDays: 270, InEA: true, Incidents: 3,
		Desc: "Group service bus."},
	{Short: "api-gateway", Name: "API Gateway", EAID: "APP-0231", Team: "integration-services",
		MUs: []string{"central", "north", "south"}, Caps: []string{"api-management"}, Lifecycle: "production",
		Repos:     []Repo{r("gateway-config", true, 4, 2, "Routes and policies of the API gateway.")},
		Workloads: []Workload{w("integration-prod", "gateway", true)},
		ITSMName:  "API Gateway", EAStaleDays: 100, InEA: true,
		Desc: "Group API gateway."},
	{Short: "document-service", Name: "Document Service", EAID: "APP-0232", Team: "integration-services",
		MUs: []string{"central"}, Caps: []string{"document-management"}, Lifecycle: "production",
		Repos:     []Repo{r("document-service", true, 3, 13, "Generates, stores and delivers documents.")},
		Workloads: []Workload{w("integration-prod", "document-service", true)},
		ITSMName:  "Document Service", EAStaleDays: 340, InEA: true, Incidents: 1,
		Desc: "Document generation and delivery."},
	{Short: "document-archive", Name: "Document Archive", EAID: "APP-0233", Team: "integration-services",
		MUs: []string{"north"}, Caps: []string{"document-management"}, Lifecycle: "production",
		Repos:     []Repo{r("doc-archive", true, 1, 80, "Long-term archive for customer documents.")},
		Workloads: []Workload{w("integration-prod", "doc-archive", true)},
		ITSMName:  "Document Archive", EAStaleDays: 610, InEA: true,
		Desc: "Long-term document archive."},
	// the integration platform market unit South runs on its own: not in the EA repository
	{Short: "south-integration-hub", Name: "south-integration-hub", Team: "south-digital",
		MUs: []string{"south"}, Lifecycle: "production",
		Repos: []Repo{{Name: "south-camel-hub", Committers: 2, PushedDays: 4, Readme: "Integration routes for Kestrel South.", Team: "south-digital"}},
		Workloads: []Workload{
			{NS: "south-int", Name: "camel-hub", Label: true, Type: "Deployment", Repo: "south-camel-hub"},
			{NS: "south-int", Name: "camel-hub-admin", Label: true, Type: "Deployment"},
		},
		Endpoints: []Endpoint{{"int.south.kestrel.example", "/", "internal", "", "camel-hub"}},
		InEA:      false,
		Desc:      "Integration platform of Kestrel South."},
	// identity-access
	{Short: "identity-hub", Name: "Identity Hub", EAID: "APP-0240", Team: "identity-and-access",
		MUs: []string{"central", "north", "south"}, Caps: []string{"identity-and-access-management"}, Lifecycle: "production",
		Repos:     []Repo{r("identity-hub", true, 5, 2, "Customer and employee login, OIDC provider.")},
		Workloads: []Workload{w("identity-prod", "identity-hub", true)},
		Endpoints: []Endpoint{{"login.kestrel.example", "/", "internet", "enforced", "identity-hub"}},
		ITSMName:  "Identity Hub", EAStaleDays: 45, InEA: true, Incidents: 2,
		Desc: "Login for customers and employees."},
	// south-digital
	{Short: "south-portal", Name: "Kestrel South Customer Portal", EAID: "APP-0250", Team: "south-digital",
		MUs: []string{"south"}, Caps: []string{"customer-self-service"}, Lifecycle: "production",
		Repos:     []Repo{r("south-portal", true, 4, 3, "Customer portal for Kestrel South.")},
		Workloads: []Workload{w("south-prod", "south-portal", true)},
		Endpoints: []Endpoint{{"www.kestrel-south.example", "/", "internet", "monitor", "south-portal"}},
		Deps:      []string{"identity-hub", "payment-gateway"},
		ITSMName:  "South Customer Portal", EAStaleDays: 260, InEA: true, Incidents: 1,
		Desc: "Customer portal for Kestrel South."},
	{Short: "south-claims", Name: "South Claims", EAID: "APP-0251", Team: "south-digital",
		MUs: []string{"south"}, Caps: []string{"claims-intake"}, Lifecycle: "production",
		Repos:     []Repo{r("south-claims", true, 3, 8, "Claims handling for Kestrel South.")},
		Workloads: []Workload{w("south-prod", "south-claims", true)},
		Deps:      []string{"payment-gateway"},
		ITSMName:  "South Claims", EAStaleDays: 380, InEA: true,
		Desc: "Claims handling for Kestrel South."},
	// platform-engineering
	{Short: "ci-platform", Name: "CI Platform", EAID: "APP-0260", Team: "platform-engineering",
		MUs: []string{"central"}, Lifecycle: "production",
		Repos:     []Repo{r("ci-templates", true, 6, 1, "Shared pipeline templates and runners.")},
		Workloads: []Workload{w("platform-prod", "runner-controller", true)},
		ITSMName:  "CI Platform", EAStaleDays: 200, InEA: true,
		Desc: "Build and deployment pipelines."},
	{Short: "observability", Name: "Observability Stack", EAID: "APP-0261", Team: "platform-engineering",
		MUs: []string{"central"}, Lifecycle: "production",
		Repos:     []Repo{r("obs-stack", true, 4, 3, "Metrics, logs and traces for every cluster.")},
		Workloads: []Workload{w("platform-prod", "grafana", true), w("platform-prod", "prometheus", true)},
		ITSMName:  "Observability Stack", EAStaleDays: 190, InEA: true,
		Desc: "Metrics, logs and traces."},
}

// background repositories: libraries, configuration and sandboxes that realise no application.
var background = []Repo{
	{Name: "java-commons", Committers: 9, PushedDays: 5, Readme: "Shared Java libraries.", Team: "platform-engineering"},
	{Name: "helm-charts", Committers: 7, PushedDays: 2, Readme: "Helm charts used by every team.", Team: "platform-engineering"},
	{Name: "terraform-modules", Committers: 5, PushedDays: 9, Readme: "Terraform modules for cloud accounts.", Team: "platform-engineering"},
	{Name: "design-system", Committers: 6, PushedDays: 4, Readme: "Kestrel UI components.", Team: "customer-digital"},
	{Name: "architecture-decisions", Committers: 8, PushedDays: 11, Readme: "Group architecture decision records.", Team: "platform-engineering"},
	{Name: "oidc-playground", Committers: 1, PushedDays: 120, Readme: "Experiments with OIDC flows.", Team: "identity-and-access"},
	{Name: "risk-sandbox", Committers: 2, PushedDays: 200, Readme: "Scratch notebooks for model ideas.", Team: "risk-analytics"},
	{Name: "doc-templates-old", Committers: 0, PushedDays: 900, Readme: "Old document templates, kept for reference.", Team: "integration-services"},
	{Name: "claims-data-export", Committers: 1, PushedDays: 70, Readme: "One-off export scripts for claims data.", Team: "claims-platform"},
	{Name: "south-infra", Committers: 2, PushedDays: 6, Readme: "Infrastructure code for Kestrel South.", Team: "south-digital"},
	{Name: "dbt-models", Committers: 4, PushedDays: 3, Readme: "Shared dbt models.", Team: "data-platform"},
	{Name: "load-tests", Committers: 3, PushedDays: 33, Readme: "Load tests for customer-facing services.", Team: "platform-engineering"},
}

type Decision struct {
	Number     int
	Title      string
	Status     string
	Days       int
	Constrains []string // application short names
	Supersedes int
}

var decisions = []Decision{
	{1, "Use the Identity Hub for every customer login", "accepted", 600, []string{"identity-hub", "customer-portal", "south-portal"}, 0},
	{2, "Retire Quote Engine Classic in favour of the Rating Engine", "accepted", 500, []string{"quote-classic", "rating-engine"}, 0},
	{3, "All customer payments go through the Payment Gateway", "accepted", 420, []string{"payment-gateway", "payhub", "collections"}, 0},
	{4, "Internet endpoints sit behind the WAF in enforce mode", "accepted", 380, []string{"api-gateway"}, 0},
	{5, "Asynchronous integration runs on the Enterprise Service Bus", "accepted", 350, []string{"esb", "document-service"}, 0},
	{6, "Point-to-point REST between claims services", "superseded", 700, []string{"claims-intake", "claims-settlement"}, 0},
	{7, "Claims events are published on the Enterprise Service Bus", "accepted", 200, []string{"claims-intake", "claims-settlement", "esb"}, 6},
	{8, "PostgreSQL is the default relational database", "accepted", 150, []string{"data-warehouse", "policy-admin"}, 0},
}

// judgements the inference pass will need, recorded once from a model.
type Judgement struct {
	Candidate  string   `json:"candidate"`
	Match      string   `json:"match"`
	Relation   string   `json:"relation"`
	Confidence float64  `json:"confidence"`
	Reasons    []string `json:"reasons"`
}

var judgements = []Judgement{
	{"githost:repositories/kestrel/pol-adm-ui", "ctx:Application/policy-admin", "realised_by", 0.82, []string{
		"README describes the policy administration frontend",
		"same owning team as pol-adm-svc, already linked by key",
		"deploys to claims-prod next to policy-admin"}},
	{"githost:repositories/kestrel/claims-intake-web", "ctx:Application/claims-intake", "realised_by", 0.95, []string{
		"README describes the online form for reporting a claim",
		"name extends claims-intake, same owning team"}},
	{"itsm:business_services/BS-1042", "ctx:Application/policy-admin", "same_as", 0.93, []string{
		"PAS is the usual abbreviation of Policy Administration System",
		"support group Claims Platform owns Policy Administration",
		"incident texts mention pol-adm-svc"}},
	{"cluster:workloads/policy-prod/policy-batch-nightly", "ctx:Application/policy-batch", "deployed_as", 0.71, []string{
		"cron job name starts with policy-batch",
		"runs in policy-prod, owned by Policy Core, which owns Policy Batch"}},
	{"githost:repositories/kestrel/doc-templates-old", "ctx:Application/document-service", "realised_by", 0.64, []string{
		"README mentions document templates",
		"same owning team as Document Service"}},
	{"githost:repositories/kestrel/risk-sandbox", "ctx:Application/risk-models", "realised_by", 0.41, []string{
		"same owning team as Risk Models",
		"README mentions models, but only as experiments"}},
}

func appByShort(s string) *App {
	for i := range apps {
		if apps[i].Short == s {
			return &apps[i]
		}
	}
	log.Fatalf("unknown app %q", s)
	return nil
}

func teamBySlug(s string) Team {
	for _, t := range teams {
		if t.Slug == s {
			return t
		}
	}
	log.Fatalf("unknown team %q", s)
	return Team{}
}

func capBySlug(s string) Cap {
	for _, c := range caps {
		if c.Slug == s {
			return c
		}
	}
	log.Fatalf("unknown capability %q", s)
	return Cap{}
}

func muBySlug(s string) MU {
	for _, m := range mus {
		if m.Slug == s {
			return m
		}
	}
	log.Fatalf("unknown market unit %q", s)
	return MU{}
}

// namespaces: owner label per namespace. claims-prod has none (its owner can be derived); batch-legacy has
// none and nothing that could tell.
var namespaces = []struct {
	Name, Cluster, Owner string
}{
	{"claims-prod", "cl-a", ""},
	{"policy-prod", "cl-a", "policy-core"},
	{"payments-prod", "cl-a", "payments"},
	{"customer-prod", "cl-a", "customer-digital"},
	{"broker-prod", "cl-a", "broker-channel"},
	{"finance-prod", "cl-a", "finance-systems"},
	{"risk-prod", "cl-a", "risk-analytics"},
	{"data-prod", "cl-a", "data-platform"},
	{"integration-prod", "cl-a", "integration-services"},
	{"identity-prod", "cl-a", "identity-and-access"},
	{"platform-prod", "cl-a", "platform-engineering"},
	{"batch-legacy", "cl-a", ""},
	{"south-prod", "cl-b", "south-digital"},
	{"south-int", "cl-b", "south-digital"},
}

func main() {
	selfCheck()
	root := "seed"
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	writeEA(root)
	writeITSM(root)
	writeGitHost(root)
	writeCluster(root)
	writeGateway(root)
	writeWAF(root)
	writeJudgements(root)
	writeTruth(root)
	fmt.Println("seed written to", root)
}

func mkdir(p string) {
	if err := os.MkdirAll(p, 0o755); err != nil {
		log.Fatal(err)
	}
}

func writeCSV(path string, header []string, rows [][]string) {
	mkdir(filepath.Dir(path))
	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	cw := csv.NewWriter(f)
	_ = cw.Write(header)
	_ = cw.WriteAll(rows)
}

func writeJSON(path string, v any) {
	mkdir(filepath.Dir(path))
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		log.Fatal(err)
	}
}

func writeYAML(path string, v any, header string) {
	mkdir(filepath.Dir(path))
	b, err := yaml.Marshal(v)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(path, append([]byte(header), b...), 0o644); err != nil {
		log.Fatal(err)
	}
}

func writeEA(root string) {
	dir := filepath.Join(root, "earepo")
	var rows [][]string
	for _, a := range apps {
		if !a.InEA {
			continue
		}
		owner := ""
		if !a.EAOwnerBlank && a.Team != "" {
			owner = teamBySlug(a.Team).Name
		}
		var muIDs, capIDs, repos []string
		for _, m := range a.MUs {
			muIDs = append(muIDs, muBySlug(m).EAID)
		}
		for _, c := range a.Caps {
			capIDs = append(capIDs, capBySlug(c).EAID)
		}
		for _, rp := range a.Repos {
			if rp.Key { // the EA tool knows the repositories somebody once entered
				repos = append(repos, "kestrel/"+rp.Name)
			}
		}
		// planted: Regulatory Reporting also claims the finance report templates repository
		if a.Short == "reg-reporting" || a.Short == "fin-reporting" {
			repos = append(repos, "kestrel/report-templates")
		}
		succ := ""
		if a.Successor != "" {
			succ = appByShort(a.Successor).EAID
		}
		rows = append(rows, []string{a.EAID, a.Name, a.Short, a.Lifecycle, owner,
			strings.Join(muIDs, ";"), strings.Join(capIDs, ";"), strings.Join(repos, ";"), succ, a.Desc,
			ts(daysAgo(a.EAStaleDays))})
	}
	writeCSV(filepath.Join(dir, "applications.csv"),
		[]string{"app_id", "name", "short_name", "lifecycle", "owner", "market_units", "capabilities", "repositories", "successor", "description", "last_modified"}, rows)

	rows = nil
	for i, c := range caps {
		parent := ""
		level := "1"
		if c.Parent != "" {
			parent = capBySlug(c.Parent).EAID
			level = "2"
		}
		rows = append(rows, []string{c.EAID, c.Name, parent, level, ts(daysAgo(300 + i*7))})
	}
	writeCSV(filepath.Join(dir, "capabilities.csv"), []string{"cap_id", "name", "parent_id", "level", "last_modified"}, rows)

	rows = nil
	for _, m := range mus {
		rows = append(rows, []string{m.EAID, m.Name, ts(daysAgo(800))})
	}
	writeCSV(filepath.Join(dir, "market_units.csv"), []string{"mu_id", "name", "last_modified"}, rows)

	rows = nil
	for _, t := range teams {
		rows = append(rows, []string{t.EAID, t.Name, muBySlug(t.MU).EAID, ts(daysAgo(240))})
	}
	writeCSV(filepath.Join(dir, "teams.csv"), []string{"team_id", "name", "market_unit", "last_modified"}, rows)

	rows = nil
	for _, a := range apps {
		if !a.InEA {
			continue
		}
		for _, d := range a.Deps {
			rows = append(rows, []string{a.EAID, appByShort(d).EAID, "sync", ts(daysAgo(a.EAStaleDays))})
		}
	}
	writeCSV(filepath.Join(dir, "interfaces.csv"), []string{"from_app", "to_app", "type", "last_modified"}, rows)
}

type businessService struct {
	SysID        string `json:"sys_id"`
	Name         string `json:"name"`
	EAReference  string `json:"ea_reference,omitempty"`
	SupportGroup string `json:"support_group"`
	Status       string `json:"operational_status"`
	Description  string `json:"description"`
	UpdatedOn    string `json:"sys_updated_on"`
}

type incident struct {
	Number          string `json:"number"`
	BusinessService string `json:"business_service"`
	OpenedAt        string `json:"opened_at"`
	Priority        string `json:"priority"`
	Short           string `json:"short_description"`
}

func writeITSM(root string) {
	dir := filepath.Join(root, "itsm")
	var bss []businessService
	var incs []incident
	n := 1000
	inc := 4100
	for _, a := range apps {
		if a.ITSMName == "" {
			continue
		}
		n++
		id := fmt.Sprintf("BS-%d", n)
		if a.Short == "policy-admin" {
			id = "BS-1042"
		}
		bs := businessService{SysID: id, Name: a.ITSMName, SupportGroup: teamBySlug(a.Team).Name + " Support",
			Status: "operational", Description: a.Desc, UpdatedOn: ts(daysAgo(10 + n%50))}
		if a.Short == "policy-admin" {
			bs.Description = "Policy Administration System"
		}
		if !a.ITSMNoRef {
			bs.EAReference = a.EAID
		}
		bss = append(bss, bs)
		for i := 0; i < a.Incidents; i++ {
			inc++
			short := "Degraded response times"
			if a.Short == "policy-admin" && i == 0 {
				short = "pol-adm-svc returns 503 after deployment"
			}
			incs = append(incs, incident{Number: fmt.Sprintf("INC%07d", inc), BusinessService: id,
				OpenedAt: ts(daysAgo(3 + i*17)), Priority: []string{"P2", "P3", "P4"}[i%3], Short: short})
		}
	}
	writeJSON(filepath.Join(dir, "business_services.json"), bss)
	writeJSON(filepath.Join(dir, "incidents.json"), incs)
}

type ghRepo struct {
	FullName      string            `json:"full_name"`
	Name          string            `json:"name"`
	Team          string            `json:"team"`
	Description   string            `json:"description"`
	Readme        string            `json:"readme_excerpt"`
	Archived      bool              `json:"archived"`
	PushedAt      string            `json:"pushed_at"`
	Committers12m int               `json:"committers_12m"`
	Catalog       map[string]string `json:"catalog_info,omitempty"`
}

type ghTeam struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ghDecision struct {
	Repository string   `json:"repository"`
	Path       string   `json:"path"`
	Number     int      `json:"number"`
	Title      string   `json:"title"`
	Status     string   `json:"status"`
	Date       string   `json:"date"`
	Constrains []string `json:"constrains"`
	Supersedes int      `json:"supersedes,omitempty"`
	URL        string   `json:"url"`
	UpdatedAt  string   `json:"updated_at"`
}

func writeGitHost(root string) {
	dir := filepath.Join(root, "githost")
	var repos []ghRepo
	add := func(rp Repo, owner, ea string) {
		team := rp.Team
		if team == "" {
			team = owner
		}
		g := ghRepo{FullName: "kestrel/" + rp.Name, Name: rp.Name, Team: team, Description: rp.Readme,
			Readme: rp.Readme, PushedAt: ts(daysAgo(rp.PushedDays)), Committers12m: rp.Committers,
			Archived: rp.PushedDays > 600}
		if rp.Key && ea != "" {
			g.Catalog = map[string]string{"ea_application_id": ea}
		}
		repos = append(repos, g)
	}
	for _, a := range apps {
		for _, rp := range a.Repos {
			add(rp, a.Team, a.EAID)
		}
	}
	for _, rp := range background {
		add(rp, "", "")
	}
	sort.Slice(repos, func(i, j int) bool { return repos[i].FullName < repos[j].FullName })
	writeJSON(filepath.Join(dir, "repositories.json"), repos)

	var ts2 []ghTeam
	for _, t := range teams {
		ts2 = append(ts2, ghTeam{Slug: t.Slug, Name: t.Name, Description: t.Name + " team"})
	}
	writeJSON(filepath.Join(dir, "teams.json"), ts2)

	var ds []ghDecision
	for _, d := range decisions {
		var cons []string
		for _, s := range d.Constrains {
			cons = append(cons, appByShort(s).EAID)
		}
		path := fmt.Sprintf("docs/adr/%04d-%s.md", d.Number, slug(d.Title))
		ds = append(ds, ghDecision{Repository: "kestrel/architecture-decisions", Path: path, Number: d.Number,
			Title: d.Title, Status: d.Status, Date: daysAgo(d.Days).Format("2006-01-02"), Constrains: cons,
			Supersedes: d.Supersedes, URL: "https://git.kestrel.example/kestrel/architecture-decisions/blob/main/" + path,
			UpdatedAt: ts(daysAgo(d.Days / 2))})
	}
	writeJSON(filepath.Join(dir, "decisions.json"), ds)
}

func slug(s string) string {
	var b strings.Builder
	dash := false
	for _, c := range strings.ToLower(s) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			b.WriteRune(c)
			dash = false
		} else if !dash {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

type k8sList struct {
	APIVersion string    `yaml:"apiVersion"`
	Kind       string    `yaml:"kind"`
	Items      []k8sItem `yaml:"items"`
}

type k8sMeta struct {
	Name              string            `yaml:"name"`
	Namespace         string            `yaml:"namespace,omitempty"`
	Labels            map[string]string `yaml:"labels,omitempty"`
	CreationTimestamp string            `yaml:"creationTimestamp"`
}

type k8sItem struct {
	APIVersion string         `yaml:"apiVersion"`
	Kind       string         `yaml:"kind"`
	Metadata   k8sMeta        `yaml:"metadata"`
	Spec       map[string]any `yaml:"spec,omitempty"`
}

func writeCluster(root string) {
	byCluster := map[string]*[2]k8sList{}
	get := func(c string) *[2]k8sList {
		if byCluster[c] == nil {
			byCluster[c] = &[2]k8sList{{APIVersion: "v1", Kind: "List"}, {APIVersion: "v1", Kind: "List"}}
		}
		return byCluster[c]
	}
	nsCluster := map[string]string{}
	for i, ns := range namespaces {
		nsCluster[ns.Name] = ns.Cluster
		labels := map[string]string{"kubernetes.io/metadata.name": ns.Name}
		if ns.Owner != "" {
			labels["team.kestrel.example/owner"] = ns.Owner
		}
		l := get(ns.Cluster)
		l[0].Items = append(l[0].Items, k8sItem{APIVersion: "v1", Kind: "Namespace",
			Metadata: k8sMeta{Name: ns.Name, Labels: labels, CreationTimestamp: ts(daysAgo(900 - i*20))}})
	}
	for _, a := range apps {
		for i, wl := range a.Workloads {
			labels := map[string]string{"app.kubernetes.io/name": wl.Name}
			if wl.Label {
				id := a.EAID
				if !a.InEA {
					id = a.Short
				}
				labels["app.kestrel.example/application-id"] = id
			}
			repo := wl.Repo
			if repo == "" && len(a.Repos) > 0 && wl.Label && !wl.NoSource {
				repo = a.Repos[min(i, len(a.Repos)-1)].Name
			}
			if repo != "" {
				labels["source.kestrel.example/repository"] = "kestrel/" + repo
			}
			image := "registry.kestrel.example/" + wl.Name + ":1." + fmt.Sprint(len(wl.Name)%9)
			spec := map[string]any{"replicas": 2, "image": image}
			if wl.Type == "CronJob" {
				spec = map[string]any{"schedule": "0 2 * * *", "image": image}
			}
			l := get(nsCluster[wl.NS])
			l[1].Items = append(l[1].Items, k8sItem{APIVersion: "apps/v1", Kind: wl.Type,
				Metadata: k8sMeta{Name: wl.Name, Namespace: wl.NS, Labels: labels, CreationTimestamp: ts(daysAgo(200 + len(wl.Name)*3))},
				Spec:     spec})
		}
	}
	header := "# kubectl get %s -A -o yaml (trimmed), cluster %s\n"
	for c, l := range byCluster {
		writeYAML(filepath.Join(root, "cluster", c, "namespaces.yaml"), l[0], fmt.Sprintf(header, "namespaces", c))
		writeYAML(filepath.Join(root, "cluster", c, "workloads.yaml"), l[1], fmt.Sprintf(header, "deployments,cronjobs", c))
	}
}

type route struct {
	ID         string            `yaml:"id"`
	Host       string            `yaml:"host"`
	Path       string            `yaml:"path"`
	Upstream   map[string]string `yaml:"upstream"`
	Visibility string            `yaml:"visibility"`
	UpdatedAt  string            `yaml:"updated_at"`
}

type wafPolicy struct {
	PolicyID  string `json:"policy_id"`
	Host      string `json:"host"`
	Path      string `json:"path"`
	Mode      string `json:"mode"`
	UpdatedAt string `json:"updated_at"`
}

func writeGateway(root string) {
	var routes []route
	n := 0
	for _, a := range apps {
		for _, e := range a.Endpoints {
			n++
			ns := a.Workloads[0].NS
			vis := "public"
			if e.Exposure == "internal" {
				vis = "internal"
			}
			routes = append(routes, route{ID: fmt.Sprintf("rt-%03d", n), Host: e.Host, Path: e.Path,
				Upstream: map[string]string{"namespace": ns, "service": e.Upstream}, Visibility: vis,
				UpdatedAt: ts(daysAgo(5 + n*3))})
		}
	}
	writeYAML(filepath.Join(root, "gateway", "routes.yaml"), map[string]any{"routes": routes}, "# API gateway route export\n")
}

func writeWAF(root string) {
	var ps []wafPolicy
	n := 0
	for _, a := range apps {
		for _, e := range a.Endpoints {
			if e.WAF == "" {
				continue
			}
			n++
			ps = append(ps, wafPolicy{PolicyID: fmt.Sprintf("waf-%03d", n), Host: e.Host, Path: e.Path, Mode: e.WAF,
				UpdatedAt: ts(daysAgo(20 + n*5))})
		}
	}
	writeJSON(filepath.Join(root, "waf", "policies.json"), ps)
}

func writeJudgements(root string) {
	f, err := os.Create(filepath.Join(root, "judgements.jsonl"))
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	for _, j := range judgements {
		b, _ := json.Marshal(j)
		f.Write(append(b, '\n'))
	}
}

// ---- the answer key ----

type truthEdge struct {
	Kind string `yaml:"kind"`
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

type truthEntity struct {
	ID      string   `yaml:"id"`
	Name    string   `yaml:"name"`
	Aliases []string `yaml:"aliases,omitempty"`
}

type finding struct {
	Insight  string   `yaml:"insight"`
	Entities []string `yaml:"entities"`
	Note     string   `yaml:"note"`
}

type truth struct {
	Note      string        `yaml:"note"`
	Reference string        `yaml:"reference_time"`
	Entities  []truthEntity `yaml:"entities"`
	Edges     []truthEdge   `yaml:"edges"`
	Findings  []finding     `yaml:"findings"`
	NotLinked []truthEdge   `yaml:"not_linked"`
}

func appID(s string) string { return "ctx:Application/" + s }

// nsOwner is who really owns a namespace: its label, or for claims-prod (no label) the claims team.
func nsOwner(ns string) string {
	if ns == "claims-prod" {
		return "claims-platform"
	}
	for _, n := range namespaces {
		if n.Name == ns {
			return n.Owner
		}
	}
	return ""
}
func repoID(s string) string { return "ctx:Repository/kestrel/" + s }
func teamID(s string) string { return "ctx:Team/" + s }

func writeTruth(root string) {
	t := truth{Note: "What the Kestrel Group really has. Generated by seed/gen; used by itx verify.", Reference: ts(Ref)}
	var es []truthEdge
	add := func(k, f, to string) { es = append(es, truthEdge{k, f, to}) }
	for _, m := range mus {
		t.Entities = append(t.Entities, truthEntity{ID: "ctx:MarketUnit/" + m.Slug, Name: m.Name, Aliases: []string{"earepo:market_units/" + m.EAID}})
	}
	for _, tm := range teams {
		t.Entities = append(t.Entities, truthEntity{ID: teamID(tm.Slug), Name: tm.Name})
		add("part_of", teamID(tm.Slug), "ctx:MarketUnit/"+tm.MU)
	}
	for _, c := range caps {
		t.Entities = append(t.Entities, truthEntity{ID: "ctx:Capability/" + c.Slug, Name: c.Name, Aliases: []string{"earepo:capabilities/" + c.EAID}})
		if c.Parent != "" {
			add("refines", "ctx:Capability/"+c.Slug, "ctx:Capability/"+c.Parent)
		}
	}
	bsN := 1000
	for _, a := range apps {
		e := truthEntity{ID: appID(a.Short), Name: a.Name}
		if a.InEA {
			e.Aliases = append(e.Aliases, "earepo:applications/"+a.EAID)
		}
		if a.ITSMName != "" {
			bsN++
			id := fmt.Sprintf("BS-%d", bsN)
			if a.Short == "policy-admin" {
				id = "BS-1042"
			}
			e.Aliases = append(e.Aliases, "itsm:business_services/"+id)
		}
		t.Entities = append(t.Entities, e)
		if a.Team != "" {
			add("owned_by", appID(a.Short), teamID(a.Team))
		}
		for _, m := range a.MUs {
			add("operated_by", appID(a.Short), "ctx:MarketUnit/"+m)
		}
		for _, c := range a.Caps {
			add("supports", appID(a.Short), "ctx:Capability/"+c)
		}
		for _, d := range a.Deps {
			add("depends_on", appID(a.Short), appID(d))
		}
		if a.Successor != "" {
			add("replaced_by", appID(a.Short), appID(a.Successor))
		}
		for _, rp := range a.Repos {
			add("realised_by", appID(a.Short), repoID(rp.Name))
			team := rp.Team
			if team == "" {
				team = a.Team
			}
			add("owned_by", repoID(rp.Name), teamID(team))
		}
		for i, wl := range a.Workloads {
			wid := "ctx:Workload/" + wl.NS + "/" + wl.Name
			add("deployed_as", appID(a.Short), wid)
			add("runs_in", wid, "ctx:Namespace/"+wl.NS)
			add("owned_by", wid, teamID(nsOwner(wl.NS)))
			repo := wl.Repo
			if repo == "" && len(a.Repos) > 0 && wl.Label && !wl.NoSource {
				repo = a.Repos[min(i, len(a.Repos)-1)].Name
			}
			if repo != "" {
				add("built_from", wid, repoID(repo))
			}
		}
		for _, ep := range a.Endpoints {
			eid := "ctx:Endpoint/" + ep.Host + ep.Path
			add("exposed_via", appID(a.Short), eid)
			add("routes_to", eid, "ctx:Workload/"+a.Workloads[0].NS+"/"+ep.Upstream)
		}
	}
	for _, rp := range background {
		add("owned_by", repoID(rp.Name), teamID(rp.Team))
	}
	for _, ns := range namespaces {
		owner := ns.Owner
		if ns.Name == "claims-prod" {
			owner = "claims-platform"
		}
		if owner != "" {
			add("owned_by", "ctx:Namespace/"+ns.Name, teamID(owner))
		}
	}
	for _, d := range decisions {
		did := fmt.Sprintf("ctx:Decision/kestrel/architecture-decisions/%d", d.Number)
		for _, c := range d.Constrains {
			add("constrains", did, appID(c))
		}
		if d.Supersedes != 0 {
			add("supersedes", did, fmt.Sprintf("ctx:Decision/kestrel/architecture-decisions/%d", d.Supersedes))
		}
	}
	sort.Slice(es, func(i, j int) bool {
		if es[i].Kind != es[j].Kind {
			return es[i].Kind < es[j].Kind
		}
		if es[i].From != es[j].From {
			return es[i].From < es[j].From
		}
		return es[i].To < es[j].To
	})
	t.Edges = es
	// Links the resolver must not make with confidence. (The EA repository's own false claim that
	// Regulatory Reporting realises report-templates is a source error, found by conflicting_realisation.)
	t.NotLinked = []truthEdge{
		{"realised_by", appID("document-service"), repoID("doc-templates-old")},
		{"realised_by", appID("risk-models"), repoID("risk-sandbox")},
		{"realised_by", appID("identity-hub"), repoID("oidc-playground")},
	}
	t.Findings = planted()
	writeYAML(filepath.Join(root, "truth.yaml"), t, "# The answer key for the Kestrel Group seed. Generated by seed/gen. Do not edit.\n")
}

// planted lists the problems hidden in the seed, as the insights that must find them.
func planted() []finding {
	return []finding{
		{"unowned_production_app", []string{appID("solvency-calc")}, "EA record has no owner; it can only be derived from the namespace"},
		{"ghost_application", []string{appID("market-insight")}, "EA says production; no repository, workload or incident anywhere"},
		{"retired_but_running", []string{appID("quote-classic")}, "EA says retired; a workload still runs in policy-prod"},
		{"exposed_without_waf", []string{"ctx:Endpoint/brokers.kestrel.example/"}, "internet endpoint without a WAF policy"},
		{"duplicate_capability_support", []string{"ctx:Capability/payment-processing", "ctx:MarketUnit/central", appID("payment-gateway"), appID("payhub")}, "two platforms for payment processing in Kestrel Central"},
		{"duplicate_capability_support", []string{"ctx:Capability/underwriting", "ctx:MarketUnit/central", appID("quote-classic"), appID("rating-engine")}, "the retired engine still counts until it is switched off"},
		{"truck_factor_one", []string{repoID("rating-engine")}, "one committer in twelve months behind a production application"},
		{"truck_factor_one", []string{repoID("doc-archive")}, "one committer in twelve months behind a production application"},
		{"conflicting_realisation", []string{repoID("report-templates")}, "two applications claim the same repository"},
		{"not_in_ea", []string{appID("south-integration-hub")}, "an integration platform group IT does not know about"},
		{"namespace_without_owner", []string{"ctx:Namespace/batch-legacy"}, "no owner label and nothing that could tell"},
	}
}

// selfCheck fails when the truth model contains problems nobody planted: an accidental duplicate
// capability or truck factor would make the answer key lie.
func selfCheck() {
	for _, c := range caps {
		if c.Slug != slug(c.Name) {
			log.Fatalf("capability %s: slug must be %s", c.Slug, slug(c.Name))
		}
	}
	for _, m := range mus {
		if m.Slug != slug(m.Name) {
			log.Fatalf("market unit %s: slug must be %s", m.Slug, slug(m.Name))
		}
	}
	for _, t := range teams {
		if t.Slug != slug(t.Name) {
			log.Fatalf("team %s: slug must be %s", t.Slug, slug(t.Name))
		}
	}
	want := map[string]bool{}
	for _, f := range planted() {
		if f.Insight == "duplicate_capability_support" {
			want[f.Entities[0]+" "+f.Entities[1]] = true
		}
	}
	got := map[string][]string{}
	for _, a := range apps {
		for _, c := range a.Caps {
			for _, m := range a.MUs {
				k := "ctx:Capability/" + c + " ctx:MarketUnit/" + m
				got[k] = append(got[k], a.Short)
			}
		}
	}
	for k, v := range got {
		if len(v) > 1 && !want[k] {
			log.Fatalf("unplanted duplicate capability support: %s %v", k, v)
		}
	}
}
