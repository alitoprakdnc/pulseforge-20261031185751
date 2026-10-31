/// <reference types="cypress" />

// Cypress E2E test: login flow, dashboard rendering, and AI chat interaction
// This test assumes the application uses data-testid attributes for key UI elements
// and follows the design system defined in the project.

describe('PulseForge End‑to‑End: Login → Dashboard → AI Chat', () => {
  const username = Cypress.env('PULSEFORGE_USERNAME')
  const password = Cypress.env('PULSEFORGE_PASSWORD')

  // Helper to perform login via UI
  const uiLogin = () => {
    cy.visit('/login')

    // Ensure the login page is loaded
    cy.get('[data-testid="login-form"]', { timeout: 10000 }).should('be.visible')

    // Fill credentials
    cy.get('[data-testid="email-input"]').should('be.visible').type(username, {
      log: false,
    })
    cy.get('[data-testid="password-input"]').should('be.visible').type(password, {
      log: false,
    })

    // Submit
    cy.get('[data-testid="login-submit"]').should('be.enabled').click()
  }

  beforeEach(() => {
    // Preserve auth cookie between tests in the same spec
    Cypress.Cookies.preserveOnce('session_id', 'jwt')
  })

  afterEach(() => {
    // Clean up any leftover state
    cy.clearLocalStorage()
    cy.clearCookies()
  })

  it('should log in, display dashboard with pipelines, and interact with AI chat', () => {
    // ----- STEP 1: LOGIN -----
    uiLogin()

    // Expect a successful login response and redirect to dashboard
    cy.intercept('POST', '/api/auth/login', (req) => {
      req.reply((res) => {
        expect(res.statusCode).to.eq(200)
        expect(res.body).to.have.property('token')
      })
    }).as('loginRequest')

    cy.wait('@loginRequest')

    // Verify we are redirected to the dashboard
    cy.url({ timeout: 15000 }).should('include', '/dashboard')

    // ----- STEP 2: DASHBOARD LOAD -----
    // Stub pipelines list to have deterministic data
    cy.fixture('pipelines.json').as('pipelinesFixture')
    cy.intercept('GET', '/api/v1/pipelines*', { fixture: 'pipelines.json' }).as('getPipelines')

    // Wait for pipelines request and ensure UI renders cards
    cy.wait('@getPipelines')
    cy.get('@pipelinesFixture').then((pipelines) => {
      const expectedCount = pipelines.length
      cy.get('[data-testid="pipeline-card"]')
        .should('have.length', expectedCount)
        .first()
        .as('firstPipelineCard')
    })

    // Verify a pipeline card displays expected information
    cy.get('@firstPipelineCard')
      .within(() => {
        cy.get('[data-testid="pipeline-name"]').should('not.be.empty')
        cy.get('[data-testid="provider-badge"]').should('exist')
        cy.get('[data-testid="health-status"]').should('exist')
      })
      .click()

    // ----- STEP 3: PIPELINE DETAIL PAGE -----
    // Intercept jobs list for the selected pipeline
    cy.fixture('jobs.json').as('jobsFixture')
    cy.intercept('GET', /\/api\/v1\/pipelines\/\d+\/jobs*/, {
      fixture: 'jobs.json',
    }).as('getJobs')

    // Intercept AI suggestions endpoint
    cy.fixture('ai_suggestions.json').as('aiSuggestionsFixture')
    cy.intercept('GET', /\/api\/v1\/pipelines\/\d+\/ai-suggestions*/, {
      fixture: 'ai_suggestions.json',
    }).as('getAISuggestions')

    // Wait for both data loads
    cy.wait(['@getJobs', '@getAISuggestions'])

    // Verify job table renders
    cy.get('[data-testid="job-table"]')
      .should('be.visible')
      .within(() => {
        cy.get('tbody tr')
          .should('have.length.at.least', 1)
          .first()
          .as('firstJobRow')
      })

    // Verify AI suggestion list renders
    cy.get('[data-testid="ai-suggestion-list"]')
      .should('be.visible')
      .within(() => {
        cy.get('[data-testid="ai-suggestion-card"]')
          .should('have.length.at.least', 1)
      })

    // ----- STEP 4: OPEN AI CHAT -----
    // The AI chat is opened via a floating action button
    cy.get('[data-testid="ai-chat-fab"]').should('be.visible').click()

    // Ensure chat modal appears
    cy.get('[data-testid="ai-chat-modal"]').should('be.visible')

    // Intercept the chat request (streaming is handled as a single POST)
    cy.intercept('POST', '/api/v1/ai/chat', (req) => {
      // Validate payload structure
      expect(req.body).to.have.property('pipelineId')
      expect(req.body).to.have.property('message')
      // Respond with a mocked streaming payload (simplified)
      req.reply({
        statusCode: 200,
        body: {
          response: 'The failure was due to a timeout in the test step.',
          suggestions: [
            'Increase the timeout value for the flaky step.',
            'Add caching for the dependency download.',
          ],
        },
      })
    }).as('aiChatRequest')

    // Type a question into the chat input
    const question = 'Why did the last job fail?'
    cy.get('[data-testid="ai-chat-input"]')
      .should('be.visible')
      .type(question, { delay: 50 })

    // Send the message
    cy.get('[data-testid="ai-chat-send"]').should('be.enabled').click()

    // Wait for the request and verify UI response
    cy.wait('@aiChatRequest')
    cy.get('[data-testid="ai-chat-bubble"]')
      .should('contain.text', 'The failure was due to a timeout')
      .and('contain.text', 'Increase the timeout value')
      .and('contain.text', 'Add caching for the dependency download')

    // Verify copy‑to‑clipboard button works for the suggestion text
    cy.get('[data-testid="ai-chat-copy"]')
      .first()
      .should('be.visible')
      .click()
    // Clipboard API is not directly testable in Cypress; verify toast appears instead
    cy.get('[data-testid="toast"]')
      .should('be.visible')
      .and('contain.text', 'Copied to clipboard')

    // Close the chat modal
    cy.get('[data-testid="ai-chat-close"]').click()
    cy.get('[data-testid="ai-chat-modal"]').should('not.exist')
  })
})