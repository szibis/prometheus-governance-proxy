# Prometheus-Governance-Proxy

# WIP very early stage PoC

#### Objective:
To develop a `remote_write` proxy that can govern, analyze, and manage the influx of metrics data from a large systems and organisations within a Prometheus environment. 
The proxy aims to prevent destination storage overload by controlling data cardinality, ensuring metadata consistency (standards), and filtering out unused metrics (also obased on external resources from dashboards or alerts)
The system will also offer a plugable architecture to facilitate the addition of new governance capabilities for the inline processing of Prometheus metrics.

#### Key Features:

1. **Data Governance**:
   - Enforce standards for metrics data (naming conventions, labels, etc.) based on your organisation needs.
   - Filter out metrics that do not adhere to predefined rules or standards.
   - Reduce cardinality dropping unnecessary labels or reformating them to be aligned to standards.

2. **Analytics and Statistics**:
   - Provide real-time analytics on the metrics data flow and top offenders.
   - Expose statistics about metrics usage, such as frequency of access, last accessed time, and relevance to dashboards or alerts.

3. **Dynamic Reconfiguration**:
   - Allow for retrospective reconfiguration of stored data based on new rules or standards.
   - Implement feedback loops to adjust governance rules dynamically based on external analytics or stats provided by internal API.

4. **Performance and Scalability**:
   - Ensure the proxy can handle high throughput from hundreds or thousands source - remote_write usualy used as output from agent/agents.
   - Optimize for minimal latency and high availability.

5. **Integration with Existing Systems**:
   - Seamlessly integrate with Prometheus `remote_write` protocol - input and output.
   - Ensure compatibility with common monitoring and alerting tools.

6. **User Interface and API**:
   - Offer an API for programmatic control and integration with other systems.
   - Dynamic reloading with configuration and setup ready for Cloud Native environments.

#### Technical Components:

1. **Proxy Service**:
   - A service that intercepts `remote_write` requests and applies governance rules before forwarding the data to the storage backend.

2. **Analytics Engine**:
   - A component that processes incoming metrics to generate usage statistics and performance metrics.

3. **Configuration Management**:
   - A system to manage and update governance rules, possibly with version control and rollback capabilities.

4. **Storage Backend**:
   - Integration with a time-series database (e.g., Prometheus TSDB, VictoriaMetrics, Grafana Mimir) for storing metrics data.

5. **User Interface Stats**:
   - A dashboard for visualizing statistics from stats exposed in Prometheus compatible formats as scraper or push.

6. **API Server**:
   - An API endpoint for automated interactions and integrations.

7. **Buffering**
   - To prevent data loss, the system will implement additional buffering and retries with exponential backoff, along with configurable chunk sizes to manage throughput and latency effectively.

#### Challenges:

- **Data Volume**: Handling the sheer volume of metrics data without introducing significant latency.
- **Complexity**: Developing a flexible rule system that can accommodate various governance needs without becoming overly complex.
- **Compatibility**: Ensuring the proxy works with different versions of Prometheus, as well as other monitoring tools.
- **User Adoption**: Creating a user-friendly system that encourages adoption and proper use by developers and operators.

#### Milestones:

1. **Requirements Gathering**:
   - Define specific governance rules and standards.
   - Identify key metrics for analytics.
   - Initial plugins support like cardinality limiter

2. **Design Phase**:
   - Architect the proxy service and its components.
   - Design the API specifications.

3. **Development Phase**:
   - Implement the proxy service with basic governance features.
   - Develop the analytics engine and integrate it with the proxy.

4. **Testing and Iteration**:
   - Conduct thorough testing with simulated and real-world data.
   - Iterate based on feedback and performance metrics.

5. **Deployment and Monitoring**:
   - Deploy the proxy in a controlled environment.
   - Monitor performance and adjust as necessary.

## Running app
```bash
go run main.go --config-file=myconfig.yml
```
