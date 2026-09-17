package history

type historyScanMetrics struct {
	transcriptWalks     int
	catalogDatabaseOpen int
	planDatabaseOpens   int
	planJSONLScans      int
	planJSONScans       int
}

func (metrics *historyScanMetrics) recordTranscriptWalk() {
	if metrics != nil {
		metrics.transcriptWalks++
	}
}

func (metrics *historyScanMetrics) recordCatalogDatabaseOpen() {
	if metrics != nil {
		metrics.catalogDatabaseOpen++
	}
}

func (metrics *historyScanMetrics) recordPlanDatabaseOpen() {
	if metrics != nil {
		metrics.planDatabaseOpens++
	}
}

func (metrics *historyScanMetrics) recordPlanJSONLScan() {
	if metrics != nil {
		metrics.planJSONLScans++
	}
}

func (metrics *historyScanMetrics) recordPlanJSONScan() {
	if metrics != nil {
		metrics.planJSONScans++
	}
}
