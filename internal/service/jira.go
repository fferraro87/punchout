package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	jiraCloud "github.com/andygrunwald/go-jira/v2/cloud"
	jira "github.com/andygrunwald/go-jira/v2/onpremise"
	d "github.com/dhth/punchout/internal/domain"
)

var (
	errJIRARepliedWithEmptyWorklog = errors.New("JIRA replied with an empty worklog; something is probably wrong")
	errCouldntCreateJiraClient     = errors.New("couldn't create JIRA client")
	errCouldntFetchIssuesFromJira  = errors.New("couldn't fetch issues from JIRA")
)

type Jira struct {
	client *jira.Client
}

func NewOnPremJiraSvc(url string, token string) (Jira, error) {
	var zero Jira

	tp := jira.BearerAuthTransport{
		Token: token,
	}
	httpClient := tp.Client()

	client, err := jira.NewClient(url, httpClient)
	if err != nil {
		return zero, fmt.Errorf("%w: %s", errCouldntCreateJiraClient, err.Error())
	}

	return Jira{
		client: client,
	}, nil
}

func NewCloudJiraSvc(url string, userName string, token string) (Jira, error) {
	var zero Jira

	tp := jiraCloud.BasicAuthTransport{
		Username: userName,
		APIToken: token,
	}
	httpClient := tp.Client()

	// Using the on-premise client regardless of the user's installation type
	// The APIs between the two installation types seem to differ, but this
	// seems to be alright for punchout's use case. If this situation changes,
	// this will need to be refactored.
	// https://github.com/andygrunwald/go-jira/issues/473
	client, err := jira.NewClient(url, httpClient)
	if err != nil {
		return zero, fmt.Errorf("%w: %s", errCouldntCreateJiraClient, err.Error())
	}

	return Jira{
		client: client,
	}, nil
}

func (svc Jira) GetIssues(jql string) ([]d.Issue, int, error) {
	var zero []d.Issue

	jIssues, resp, err := svc.client.Issue.Search(context.Background(), jql, nil)
	if err != nil {
		return zero, 0, fmt.Errorf("%w: %s", errCouldntFetchIssuesFromJira, err.Error())
	}

	issues := make([]d.Issue, len(jIssues))
	for i, issue := range jIssues {
		var assignee string
		var totalSecsSpent int
		var status string
		if issue.Fields != nil {
			if issue.Fields.Assignee != nil {
				assignee = issue.Fields.Assignee.DisplayName
			}

			totalSecsSpent = issue.Fields.AggregateTimeSpent

			if issue.Fields.Status != nil {
				status = issue.Fields.Status.Name
			}
		}
		issues[i] = d.Issue{
			IssueKey:        issue.Key,
			IssueType:       issue.Fields.Type.Name,
			Summary:         issue.Fields.Summary,
			Assignee:        assignee,
			Status:          status,
			AggSecondsSpent: totalSecsSpent,
		}
	}

	return issues, resp.StatusCode, err
}

func (svc Jira) SyncWLToJIRA(ctx context.Context, entry d.WorklogEntry, comment string, timeDeltaMins int) error {
	start := entry.BeginTS

	if timeDeltaMins != 0 {
		start = start.Add(time.Minute * time.Duration(timeDeltaMins))
	}

	timeSpentSecs := int(entry.EndTS.Sub(entry.BeginTS).Seconds())

	wl := jira.WorklogRecord{
		IssueID:          entry.IssueKey,
		Started:          (*jira.Time)(&start),
		TimeSpentSeconds: timeSpentSecs,
		Comment:          comment,
	}
	cwl, _, err := svc.client.Issue.AddWorklogRecord(ctx,
		entry.IssueKey,
		&wl,
	)

	if cwl != nil && cwl.Started == nil {
		return errJIRARepliedWithEmptyWorklog
	}
	return err
}

func (svc Jira) JiraURL() string {
	return svc.client.BaseURL.String()
}

// --- NUOVA FUNZIONE AGGIUNTA ---
// TransitionIssue cambia lo stato del task in Jira usando l'ID di transizione fornito.
func (svc Jira) TransitionIssue(ctx context.Context, issueKey string, transitionID string) error {
	// Se l'ID è vuoto, usciamo senza fare nulla per evitare errori su Jira
	if transitionID == "" {
		return nil
	}

	// Passiamo direttamente transitionID (la stringa "291") invece del payload!
	_, err := svc.client.Issue.DoTransition(ctx, issueKey, transitionID)
	if err != nil {
		return fmt.Errorf("impossibile cambiare lo stato del task %s: %w", issueKey, err)
	}

	return nil
}

// --- NUOVA FUNZIONE AGGIUNTA ---
// TransitionIssueWithEstimate cambia stato e aggiorna l'Original Estimate
func (svc Jira) TransitionIssueWithEstimate(ctx context.Context, issueKey string, transitionID string, estimate string) error {
	if transitionID == "" {
		return nil
	}

	// Creiamo un payload custom che include l'aggiornamento del time tracking
	payload := map[string]interface{}{
		"transition": map[string]interface{}{
			"id": transitionID,
		},
		"update": map[string]interface{}{
			"timetracking": []map[string]interface{}{
				{
					"edit": map[string]interface{}{
						"originalEstimate": estimate,
					},
				},
			},
		},
	}

	// Costruiamo l'URL dell'endpoint manualmente
	apiEndpoint := fmt.Sprintf("rest/api/2/issue/%s/transitions", issueKey)
	
	// Creiamo la richiesta usando il client di base
	req, err := svc.client.NewRequest(ctx, "POST", apiEndpoint, payload)
	if err != nil {
		return fmt.Errorf("errore nella creazione della richiesta per %s: %w", issueKey, err)
	}

	// Eseguiamo la richiesta
	resp, err := svc.client.Do(req, nil)
	if err != nil {
		return fmt.Errorf("errore durante la chiamata a Jira per %s: %w", issueKey, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("jira ha rifiutato la transizione (Status: %d)", resp.StatusCode)
	}

	return nil
}
