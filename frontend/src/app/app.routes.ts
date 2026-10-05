import { Routes } from '@angular/router';
import { authGuard } from './core/auth/auth.guard';
import { homeRedirectGuard } from './core/auth/home-redirect.guard';
import { roleGuard } from './core/auth/role.guard';

export const routes: Routes = [
  {
    path: '',
    loadComponent: () => import('./shared/layout/public-shell.component').then((m) => m.PublicShellComponent),
    children: [
      {
        path: '',
        loadComponent: () => import('./features/marketplace/home.component').then((m) => m.HomeComponent),
        canActivate: [homeRedirectGuard]
      },
      {
        path: 'marketplace',
        loadComponent: () => import('./features/marketplace/marketplace-list.component').then((m) => m.MarketplaceListComponent)
      },
      {
        path: 'marketplace/:id',
        loadComponent: () => import('./features/marketplace/listing-detail.component').then((m) => m.ListingDetailComponent)
      },
      { path: 'login', loadComponent: () => import('./features/auth/login.component').then((m) => m.LoginComponent) },
      { path: 'register', loadComponent: () => import('./features/auth/register.component').then((m) => m.RegisterComponent) },
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
        canActivate: [authGuard]
      },
      {
        path: 'account',
        loadComponent: () => import('./features/shared/coming-soon.component').then((m) => m.ComingSoonComponent),
        data: { titleKey: 'comingSoon.myAccount' },
        canActivate: [authGuard]
      },
      {
        path: 'account/reservations',
        loadComponent: () => import('./features/account/reservations-list.component').then((m) => m.ReservationsListComponent),
        canActivate: [authGuard]
      },
      {
        path: 'sell',
        loadComponent: () => import('./features/shared/coming-soon.component').then((m) => m.ComingSoonComponent),
        data: { titleKey: 'comingSoon.publish' },
        canActivate: [authGuard, roleGuard('farmer')]
      }
    ]
  },
  {
    path: 'portal',
    loadComponent: () => import('./shared/layout/portal-shell.component').then((m) => m.PortalShellComponent),
    canActivate: [authGuard, roleGuard('farmer')],
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
        path: 'listings',
        loadComponent: () => import('./features/shared/coming-soon.component').then((m) => m.ComingSoonComponent),
        data: { titleKey: 'portalNav.listings' }
      },
      {
        path: 'listings/new',
        loadComponent: () => import('./features/portal/listing-form.component').then((m) => m.ListingFormComponent)
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
    path: 'admin',
    loadComponent: () => import('./shared/layout/admin-shell.component').then((m) => m.AdminShellComponent),
    canActivate: [authGuard, roleGuard('admin')],
    children: [
      {
        path: '',
        loadComponent: () => import('./features/admin/admin-overview.component').then((m) => m.AdminOverviewComponent)
      },
      {
        path: 'users',
        loadComponent: () => import('./features/admin/admin-users.component').then((m) => m.AdminUsersComponent)
      },
      {
        path: 'listings',
        loadComponent: () => import('./features/admin/admin-listings.component').then((m) => m.AdminListingsComponent)
      },
      {
        path: 'escrow',
        loadComponent: () => import('./features/admin/admin-escrow.component').then((m) => m.AdminEscrowComponent)
      },
      {
        path: 'reports',
        loadComponent: () => import('./features/admin/admin-reports-list.component').then((m) => m.AdminReportsListComponent)
      },
      {
        path: 'reports/:id',
        loadComponent: () => import('./features/admin/admin-report-detail.component').then((m) => m.AdminReportDetailComponent)
      },
      {
        path: 'settings',
        loadComponent: () => import('./features/shared/coming-soon.component').then((m) => m.ComingSoonComponent),
        data: { titleKey: 'adminNav.settings' }
      }
    ]
  },
  { path: '**', redirectTo: '' }
];
