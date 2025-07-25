# dmp-helm-chart-downloader

Helm plugin to fetch resources from a helm chart

Supports the following syntax :

## Get a resource from the chart beeing installed

- chart://path/to/file.yaml => gets the file path/to/file.yaml in the chart being installed

#### Example

```
helm install myservice -f chart://values-dev.yaml myrepo/my-chart --version 1.0.2
```

gets the file "values-dev.yaml" from the current chart, myrepo/my-chart in version 1.0.2

### Dependency management

If the chart being installed has dependencies, the plugin will automatically create an aggregated values file.

#### Example 

- Suppose we are installing a chart "chart1", with a file "values-dev.yaml"
- The chart1 has a dependency called "chart2"
- The chart "chart2" has also a file "values-dev.yaml"

When you install chart1 with the option -f chart://values-dev.yaml, the plugin will return a values file as the following:

```
<content of values-dev.yaml of chart1>

chart2:
  <content of values-dev.yaml of chart2>
```

## Get a resource from another chart

- chart://path/to/file.yaml@repo/chartname[:version] => gets the file path/to/file.yaml from the chart chartname
  - "chart" is a chart pulled from the helm repo "repo"
  - "version" is the version to pull (optional)

### Example

```
helm install myservice -f chart://values-dev.yaml@config/common-conf:1.0.0 myrepo/my-chart --version 1.0.2
```

gets the file "values-dev.yaml" from the chart myrepo/my-chart in version 1.0.0

`Note`: in this case, the chart "athena/common-conf" is NOT installed. It is just pulled to extract the config file.

