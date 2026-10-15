import { Routes } from '@angular/router';
import { authGuard } from './core/auth/auth.guard';
import { guestGuard } from './core/auth/guest.guard';
import { permissionGuard, sellerGuard, staffGuard } from './core/auth/space.guards';

export const routes: Routes = [
  {
    path: '',
    loadComponent: () => import('./shared/layout/public-shell.component').then((m) => m.PublicShellComponent),
    children: [
      {
        path: '',
        loadComponent: () => import('./features/marketplace/home.component').then((m) => m.HomeComponent)
      },
      {
        path: 'marketplace',
        loadComponent: () => import('./features/marketplace/marketplace-list.component').then((m) => m.MarketplaceListComponent)
      },
      {
        path: 'marketplace/:id',
        loadComponent: () => import('./features/marketplace/listing-detail.component').then((m) => m.ListingDetailComponent),
        data: { focus: true }
      },
      {
        path: 'login',
        loadComponent: () => import('./features/auth/login.component').then((m) => m.LoginComponent),
        canActivate: [guestGuard]
      },
      {
        path: 'register',
        loadComponent: () => import('./features/auth/register.component').then((m) => m.RegisterComponent),
        canActivate: [guestGuard]
      },
      {
        path: 'forgot-password',
        loadComponent: () => import('./features/auth/password-reset.component').then((m) => m.PasswordResetComponent),
        canActivate: [guestGuard]
      },
      {
        path: 'reset-password/:token',
        loadComponent: () => import('./features/auth/password-reset.component').then((m) => m.PasswordResetComponent)
      },
      {
        // A staff invitation's link. Open to anyone holding it.
        path: 'invitation/:token',
        loadComponent: () => import('./features/auth/invitation.component').then((m) => m.InvitationComponent)
      },
      {
        path: 'account/security',
        loadComponent: () => import('./features/account/security.component').then((m) => m.AccountSecurityComponent),
        canActivate: [authGuard]
      },
      {
        path: 'terms',
        loadComponent: () => import('./features/legal/terms.component').then((m) => m.TermsComponent)
      },
      { path: 'cgu', redirectTo: 'terms' },
      {
        path: 'forum',
        loadComponent: () => import('./features/forum/forum-list.component').then((m) => m.ForumListComponent)
      },
      {
        path: 'forum/:id',
        loadComponent: () => import('./features/forum/forum-detail.component').then((m) => m.ForumDetailComponent)
      },
      {
        path: 'checkout/:listingId',
        loadComponent: () => import('./features/checkout/checkout.component').then((m) => m.CheckoutComponent),
        canActivate: [authGuard],
        data: { focus: true }
      },
      {
        path: 'account',
        loadComponent: () => import('./features/account/account.component').then((m) => m.AccountComponent),
        canActivate: [authGuard]
      },
      {
        path: 'farms/:id/reviews',
        loadComponent: () => import('./features/marketplace/farm-reviews.component').then((m) => m.FarmReviewsComponent)
      },
      {
        path: 'account/reservations/:id/review',
        loadComponent: () => import('./features/account/review-form.component').then((m) => m.ReviewFormComponent),
        canActivate: [authGuard],
        data: { focus: true }
      },
      {
        path: 'notifications',
        loadComponent: () => import('./features/notifications/notifications.component').then((m) => m.NotificationsComponent),
        canActivate: [authGuard]
      },
      {
        path: 'account/favorites',
        loadComponent: () => import('./features/account/favorites.component').then((m) => m.FavoritesComponent),
        canActivate: [authGuard]
      },
      {
        path: 'account/reservations',
        loadComponent: () => import('./features/account/reservations-list.component').then((m) => m.ReservationsListComponent),
        canActivate: [authGuard]
      },
      {
        // Open to any signed-in user: a buyer presents their farm first.
        path: 'sell',
        loadComponent: () => import('./features/sell/sell-wizard.component').then((m) => m.SellWizardComponent),
        canActivate: [authGuard],
        data: { focus: true }
      }
    ]
  },
  {
    path: 'portal',
    loadComponent: () => import('./shared/layout/portal-shell.component').then((m) => m.PortalShellComponent),
    canActivate: [authGuard, sellerGuard],
    children: [
      {
        path: '',
        loadComponent: () => import('./features/portal/portal-dashboard.component').then((m) => m.PortalDashboardComponent)
      },
      {
        path: 'cycles',
        loadComponent: () => import('./features/portal/portal-cycles-list.component').then((m) => m.PortalCyclesListComponent)
      },
      {
        path: 'cycles/new',
        loadComponent: () => import('./features/portal/cycle-form.component').then((m) => m.CycleFormComponent)
      },
      {
        path: 'cycles/:id',
        loadComponent: () => import('./features/portal/cycle-detail.component').then((m) => m.CycleDetailComponent)
      },
      {
        path: 'listings',
        loadComponent: () => import('./features/portal/portal-listings.component').then((m) => m.PortalListingsComponent)
      },
      {
        path: 'listings/new',
        loadComponent: () => import('./features/portal/listing-form.component').then((m) => m.ListingFormComponent)
      },
      {
        path: 'listings/:id',
        loadComponent: () => import('./features/portal/listing-edit.component').then((m) => m.ListingEditComponent)
      },
      {
        path: 'farm',
        loadComponent: () => import('./features/portal/farm-settings.component').then((m) => m.FarmSettingsComponent)
      },
      {
        path: 'farms',
        loadComponent: () => import('./features/portal/portal-farms.component').then((m) => m.PortalFarmsComponent)
      },
      {
        path: 'reservations',
        loadComponent: () =>
          import('./features/portal/portal-reservations-list.component').then((m) => m.PortalReservationsListComponent)
      },
      {
        path: 'payments',
        loadComponent: () => import('./features/portal/portal-payments.component').then((m) => m.PortalPaymentsComponent)
      }
    ]
  },
  {
    // The back office. `staffGuard` keeps everyone else off it; each
    // page then asks for the permission its API calls need, so a role
    // lands on the overview rather than on a page of refusals.
    path: 'admin',
    loadComponent: () => import('./shared/layout/admin-shell.component').then((m) => m.AdminShellComponent),
    canActivate: [authGuard, staffGuard],
    children: [
      {
        path: '',
        loadComponent: () => import('./features/admin/admin-overview.component').then((m) => m.AdminOverviewComponent)
      },
      {
        path: 'farms',
        loadComponent: () => import('./features/admin/admin-farms.component').then((m) => m.AdminFarmsComponent),
        canActivate: [permissionGuard('farms.read')]
      },
      {
        path: 'farms/:id',
        loadComponent: () => import('./features/admin/admin-farms.component').then((m) => m.AdminFarmsComponent),
        canActivate: [permissionGuard('farms.read')]
      },
      {
        path: 'users',
        loadComponent: () => import('./features/admin/admin-users.component').then((m) => m.AdminUsersComponent),
        canActivate: [permissionGuard('accounts.read')]
      },
      {
        path: 'listings',
        loadComponent: () => import('./features/admin/admin-listings.component').then((m) => m.AdminListingsComponent),
        canActivate: [permissionGuard('catalog.read')]
      },
      {
        path: 'escrow',
        loadComponent: () => import('./features/admin/admin-escrow.component').then((m) => m.AdminEscrowComponent),
        canActivate: [permissionGuard('escrow.read')]
      },
      {
        path: 'reports',
        loadComponent: () => import('./features/admin/admin-reports.component').then((m) => m.AdminReportsComponent),
        canActivate: [permissionGuard('reports.read')]
      },
      {
        path: 'reports/:id',
        loadComponent: () => import('./features/admin/admin-reports.component').then((m) => m.AdminReportsComponent),
        canActivate: [permissionGuard('reports.read')]
      },
      {
        path: 'disputes',
        loadComponent: () => import('./features/admin/admin-reports.component').then((m) => m.AdminReportsComponent),
        canActivate: [permissionGuard('reports.read')],
        data: { disputes: true }
      },
      {
        path: 'disputes/:id',
        loadComponent: () => import('./features/admin/admin-reports.component').then((m) => m.AdminReportsComponent),
        canActivate: [permissionGuard('reports.read')],
        data: { disputes: true }
      },
      {
        path: 'staff',
        loadComponent: () => import('./features/admin/admin-staff.component').then((m) => m.AdminStaffComponent),
        canActivate: [permissionGuard('staff.manage')]
      },
      // Settings was folded away: its switches show on the overview and
      // the team page, and what can be edited is the reference data.
      { path: 'settings', redirectTo: 'reference' },
      {
        // One route per resource, each loaded when opened.
        path: 'reference',
        loadComponent: () => import('./features/admin/admin-reference.component').then((m) => m.AdminReferenceComponent),
        canActivate: [permissionGuard('reference.manage')],
        children: [
          { path: '', pathMatch: 'full', redirectTo: 'categories' },
          {
            path: 'categories',
            loadComponent: () => import('./features/admin/reference/categories-view.component').then((m) => m.CategoriesViewComponent)
          },
          {
            path: 'places',
            loadComponent: () => import('./features/admin/reference/places-view.component').then((m) => m.PlacesViewComponent)
          },
          {
            path: 'lists',
            loadComponent: () => import('./features/admin/reference/lists-view.component').then((m) => m.ListsViewComponent)
          },
          {
            path: 'history',
            loadComponent: () => import('./features/admin/reference/history-view.component').then((m) => m.HistoryViewComponent)
          },
          {
            path: 'advanced',
            loadComponent: () => import('./features/admin/reference/advanced-view.component').then((m) => m.AdvancedViewComponent)
          }
        ]
      },
      {
        path: 'audit',
        loadComponent: () => import('./features/admin/admin-audit.component').then((m) => m.AdminAuditComponent),
        canActivate: [permissionGuard('audit.read')]
      }
    ]
  },
  { path: '**', redirectTo: '' }
];
