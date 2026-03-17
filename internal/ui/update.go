package ui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	d "github.com/dhth/punchout/internal/domain"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	m.message = ""

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if m.issueList.FilterState() == list.Filtering {
			m.issueList, cmd = m.issueList.Update(msg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		}
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			var saveCmd tea.Cmd
			var ret bool
			switch m.activeView {
			case editActiveWLView:
				saveCmd = m.getCmdToUpdateActiveWL()
				ret = true
			case saveActiveWLView:
				saveCmd = m.getCmdToSaveActiveWL()
				ret = true
			case wlEntryView:
				saveCmd = m.getCmdToSaveOrUpdateWL()
				ret = true
			case estimateEntryView: // --- INVIO DA MODALE ESTIMATE ---
				saveCmd = m.getCmdToSubmitEstimateAndStartTracking()
				ret = true
			}
			if saveCmd != nil {
				cmds = append(cmds, saveCmd)
			}
			if ret {
				return m, tea.Batch(cmds...)
			}
		case "ctrl+s":
			switch m.activeView {
			case saveActiveWLView, wlEntryView:
				m.handleRequestToSyncTimestamps()
			}
		case "esc":
			quit := m.handleEscape()
			if quit {
				return m, tea.Quit
			}
		case "tab":
			viewSwitchCmd := m.getCmdToGoForwardsInViews()
			if viewSwitchCmd != nil {
				cmds = append(cmds, viewSwitchCmd)
			}
		case "shift+tab":
			viewSwitchCmd := m.getCmdToGoBackwardsInViews()
			if viewSwitchCmd != nil {
				cmds = append(cmds, viewSwitchCmd)
			}
		case "k":
			err := m.shiftTime(shiftBackward, shiftMinute)
			if err != nil {
				return m, tea.Batch(cmds...)
			}
		case "j":
			err := m.shiftTime(shiftForward, shiftMinute)
			if err != nil {
				return m, tea.Batch(cmds...)
			}
		case "K":
			err := m.shiftTime(shiftBackward, shiftFiveMinutes)
			if err != nil {
				return m, tea.Batch(cmds...)
			}
		case "J":
			err := m.shiftTime(shiftForward, shiftFiveMinutes)
			if err != nil {
				return m, tea.Batch(cmds...)
			}
		case "h":
			err := m.shiftTime(shiftBackward, shiftDay)
			if err != nil {
				return m, tea.Batch(cmds...)
			}
		case "l":
			err := m.shiftTime(shiftForward, shiftDay)
			if err != nil {
				return m, tea.Batch(cmds...)
			}
		}
	}

	// --- AGGIORNAMENTO CAMPI DI TESTO ---
	switch m.activeView {
	case editActiveWLView, saveActiveWLView, wlEntryView:
		for i := range m.trackingInputs {
			m.trackingInputs[i], cmd = m.trackingInputs[i].Update(msg)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)
	case estimateEntryView: // Modale dell'estimate
		m.estimateInput, cmd = m.estimateInput.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			quit := m.handleRequestToGoBackOrQuit()
			if quit {
				return m, tea.Quit
			}
		case "1":
			if m.activeView != issueListView {
				m.activeView = issueListView
			}
		case "2":
			if m.activeView != wLView {
				m.activeView = wLView
				cmds = append(cmds, fetchUnsyncedWorkLogs(m.db))
			}
		case "3":
			if m.activeView != syncedWLView {
				m.activeView = syncedWLView
			}
		case "ctrl+r":
			reloadCmd := m.getCmdToReloadData()
			if reloadCmd != nil {
				cmds = append(cmds, reloadCmd)
			}
		case "ctrl+t":
			m.handleRequestToGoToActiveIssue()
		case "ctrl+s":
			if !m.issuesFetched {
				break
			}

			switch m.activeView {
			case issueListView:
				switch m.trackingActive {
				case true:
					m.handleRequestToUpdateActiveWL()
				case false:
					m.handleRequestToCreateManualWL()
				}
			case wLView:
				m.handleRequestToUpdateSavedWL()
			}

		case "u":
			if m.activeView != wLView {
				break
			}
			m.handleRequestToUpdateSavedWL()

		case "ctrl+d":
			switch m.activeView {
			case wLView:
				deleteCmd := m.getCmdToDeleteWL()
				if deleteCmd != nil {
					cmds = append(cmds, deleteCmd)
				}
			}
		case "ctrl+x":
			if m.activeView == issueListView && m.trackingActive {
				cmds = append(cmds, deleteActiveIssueLog(m.db))
			}
		case "S":
			if m.activeView != issueListView {
				break
			}
			quickSwitchCmd := m.getCmdToQuickSwitchTracking()
			if quickSwitchCmd != nil {
				cmds = append(cmds, quickSwitchCmd)
			}

		case "f":
			if m.activeView != issueListView {
				break
			}

			if !m.trackingActive {
				m.message = "nothing is being tracked right now"
				break
			}

			handleCmd := m.getCmdToQuickFinishActiveWL()
			if handleCmd != nil {
				cmds = append(cmds, handleCmd)
			}
                case "s":
			if !m.issuesFetched {
				break
			}

			switch m.activeView {
			case issueListView:
				issue, ok := m.issueList.SelectedItem().(*d.Issue)
				if !ok {
					break
				}

				isStarting := !m.trackingActive
				status := strings.TrimSpace(issue.Status)

				// 1. GESTIONE STATO "NEW" (Doppia transizione)
				if isStarting && strings.EqualFold(status, "New") {
					m.activeView = estimateEntryView
					m.estimateInput.Focus()
					m.estimateInput.SetValue("")
					break 
				}

				// Avvia/Ferma il timer locale
				handleCmd := m.getCmdToToggleTracking()
				if handleCmd != nil {
					cmds = append(cmds, handleCmd)
				}
				
				// 2. GESTIONE TRANSIZIONE A "IN PROGRESS"
				// Solo se stiamo iniziando E lo stato attuale NON è già "In Progress"
				if isStarting && !strings.EqualFold(status, "In Progress") {
					transitionCmd := m.getCmdToTransitionIssueToInProgress()
					if transitionCmd != nil {
						cmds = append(cmds, transitionCmd)
					}
				}

			case wLView:
				syncCmds := m.getCmdToSyncWLToJIRA()
				if len(syncCmds) > 0 {
					cmds = append(cmds, syncCmds...)
				}
			}
		case "?":
			if m.activeView == issueListView || m.activeView == wLView || m.activeView == syncedWLView {
				m.lastView = m.activeView
				m.activeView = helpView
			}
		case "ctrl+b":
			if !m.issuesFetched {
				break
			}

			if m.activeView == issueListView {
				cmds = append(cmds, m.getCmdToOpenIssueInBrowser())
			}
		}

	case tea.WindowSizeMsg:
		m.handleWindowResizing(msg)
		
	// --- SE HA FINITO IL DOPPIO SALTO DI STATO, AVVIA IL TIMER ---
	case newIssueReadyToTrack:
		m.message = "Transitions successful! Starting timer..."
		cmds = append(cmds, m.getCmdToStartTracking())
	// -------------------------------------------------------------

	case issueTransitionedOnJIRA:
		if msg.err != nil {
			m.message = fmt.Sprintf("Error transitioning issue %s: %v", msg.issueKey, msg.err)
			m.messages = append(m.messages, m.message)
		}

	case issuesFetchedFromJIRA:
		handleCmd := m.handleIssuesFetchedFromJIRAMsg(msg)
		if handleCmd != nil {
			cmds = append(cmds, handleCmd)
		}
	case manualWLInsertedInDB:
		handleCmd := m.handleManualEntryInsertedInDBMsg(msg)
		if handleCmd != nil {
			cmds = append(cmds, handleCmd)
		}
	case wLUpdatedInDB:
		handleCmd := m.handleWLUpdatedInDBMsg(msg)
		if handleCmd != nil {
			cmds = append(cmds, handleCmd)
		}
	case wLEntriesFetchedFromDB:
		m.handleWLEntriesFetchedFromDBMsg(msg)
	case syncedWLEntriesFetchedFromDB:
		m.handleSyncedWLEntriesFetchedFromDBMsg(msg)
	case wLSyncUpdatedInDB:
		m.handleWLSyncUpdatedInDBMsg(msg)
	case activeWLFetchedFromDB:
		m.handleActiveWLFetchedFromDBMsg(msg)
	case wLDeletedFromDB:
		handleCmd := m.handleWLDeletedFromDBMsg(msg)
		if handleCmd != nil {
			cmds = append(cmds, handleCmd)
		}
	case activeWLDeletedFromDB:
		m.handleActiveWLDeletedFromDBMsg(msg)
	case wLSyncedToJIRA:
		handleCmd := m.handleWLSyncedToJIRAMsg(msg)
		if handleCmd != nil {
			cmds = append(cmds, handleCmd)
		}
	case activeWLUpdatedInDB:
		m.handleActiveWLUpdatedInDBMsg(msg)
	case trackingToggledInDB:
		handleCmd := m.handleTrackingToggledInDBMsg(msg)
		if handleCmd != nil {
			cmds = append(cmds, handleCmd)
		}
	case activeWLSwitchedInDB:
		m.handleActiveWLSwitchedInDBMsg(msg)
	case hideHelpMsg:
		m.showHelpIndicator = false
	case urlOpenedinBrowserMsg:
		if msg.err != nil {
			m.message = fmt.Sprintf("Error opening url: %s", msg.err.Error())
		}
	}

	switch m.activeView {
	case issueListView:
		m.issueList, cmd = m.issueList.Update(msg)
		cmds = append(cmds, cmd)
	case wLView:
		m.worklogList, cmd = m.worklogList.Update(msg)
		cmds = append(cmds, cmd)
	case syncedWLView:
		m.syncedWorklogList, cmd = m.syncedWorklogList.Update(msg)
		cmds = append(cmds, cmd)
	case helpView:
		m.helpVP, cmd = m.helpVP.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}
