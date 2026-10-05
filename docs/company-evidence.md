# Company-alignment evidence

Mercury is inspired by Delivery Hero AdTech's publicly visible engineering problem space. It does not claim to reproduce proprietary architecture, topology, infrastructure, or SLOs.

Every material company-alignment statement uses one classification:

- `PUBLICLY_CONFIRMED`: supported by a cited official public source.
- `PROJECT_CHOICE`: chosen for Mercury's needs, not attributed to Delivery Hero.
- `UNKNOWN`: not supported by current public evidence; verify later if it matters.

Company adoption is never sufficient justification. A future technology should solve a measured Mercury need and, where useful, be compatible with current public evidence.

| Topic | Status | Evidence and boundary |
| --- | --- | --- |
| Go in Delivery Hero AdTech | `PUBLICLY_CONFIRMED` | Official AdTech campaign and display-ad job descriptions explicitly request Go: [Campaigns](https://careers.deliveryhero.com/job/software-engineer-ii-adtech-campaigns-vendor-in-berlin-germany-jid-10807), [Display Ads](https://careers.deliveryhero.com/job/software-engineer-golang-adtech-display-ads-all-genders-in-berlin-germany-jid-2681). This does not prescribe Mercury service topology. |
| React and TypeScript in Delivery Hero AdTech | `PUBLICLY_CONFIRMED` | An official Display Ads/Vendor role describes customer-facing React and TypeScript work: [Software Engineer (React)](https://careers.deliveryhero.com/job/software-engineer-react-display-ads-acquisitions-vendor-in-berlin-germany-jid-9844). |
| Frontend testing/API integration | `PUBLICLY_CONFIRMED` | The same role names REST/gRPC integration and unit, integration, and E2E testing. |
| Microfrontend principles in AdTech | `PUBLICLY_CONFIRMED` | The official Display Ads role mentions microfrontends and event-driven microservices. This is evidence of use, not authority to decompose Mercury prematurely: [Display Ads](https://careers.deliveryhero.com/job/software-engineer-golang-adtech-display-ads-all-genders-in-berlin-germany-jid-2681). |
| Production observability/monitoring in AdTech | `PUBLICLY_CONFIRMED` | Official AdTech roles discuss observability/monitoring and list tools such as Grafana and Prometheus: [AdTech Golang](https://careers.deliveryhero.com/job/software-engineer-golang-adtech-all-genders-in-berlin-germany-jid-2022). Tool presence does not prescribe Mercury's observability stack. |
| SPA and Redux usage in the relevant AdTech product | `UNKNOWN` | Current sources above establish React/TypeScript and microfrontend work but do not justify a more specific Redux or SPA claim. Verify if a future frontend decision depends on it. |
| Kubernetes in Delivery Hero AdTech | `PUBLICLY_CONFIRMED` | Official AdTech roles list Kubernetes experience: [Display Ads](https://careers.deliveryhero.com/job/software-engineer-golang-adtech-display-ads-all-genders-in-berlin-germany-jid-2681), [Campaigns](https://careers.deliveryhero.com/job/software-engineer-ii-adtech-campaigns-vendor-in-berlin-germany-jid-10807). Specific deployment patterns remain `UNKNOWN`. |
| GCP in Delivery Hero AdTech | `PUBLICLY_CONFIRMED` | An official AdTech Golang role names GCP with Terraform, Docker, Kubernetes, and Helm: [AdTech Golang](https://careers.deliveryhero.com/job/software-engineer-golang-adtech-all-genders-in-berlin-germany-jid-2022). Provider topology and exclusivity remain `UNKNOWN`. |
| Next.js | `PROJECT_CHOICE` | Mercury currently uses it; no relevant public company evidence is asserted. |
| Mercury's modular monolith and PostgreSQL-first baseline | `PROJECT_CHOICE` | Chosen to establish correctness and produce evidence before decomposition. |
| Specific service boundaries, autoscaler, readiness design, service mesh, clusters, regions, SLOs | `UNKNOWN` | No proprietary details are inferred from job descriptions. |

Evidence was rechecked on 2026-10-06. Revalidate time-sensitive sources when a later phase relies on them.
