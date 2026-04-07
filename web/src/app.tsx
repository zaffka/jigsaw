import { Route, Switch, Redirect } from 'wouter';
import { I18nProvider } from './i18n';
import { Login } from './pages/Login';
import { AdminLayout } from './pages/admin/AdminLayout';
import { CatalogList } from './pages/admin/CatalogList';
import { CatalogCreate } from './pages/admin/CatalogCreate';
import { CatalogEdit } from './pages/admin/CatalogEdit';
import { UserList } from './pages/admin/UserList';
import { CatalogPublic } from './pages/catalog/CatalogPublic';
import { GameScreen } from './pages/game/GameScreen';
import { RewardPage } from './pages/game/RewardPage';
import { About } from './pages/About';
import { ParentLayout } from './pages/parent/ParentLayout';
import { PuzzleList } from './pages/parent/PuzzleList';
import { PuzzleDetail } from './pages/parent/PuzzleDetail';
import { ChildList } from './pages/parent/ChildList';

export function App() {
  return (
    <I18nProvider>
      <Switch>
        <Route path="/login" component={Login} />

        <Route path="/admin/catalog/new">
          <AdminLayout>
            <CatalogCreate />
          </AdminLayout>
        </Route>

        <Route path="/admin/catalog/:id/edit">
          <AdminLayout>
            <CatalogEdit />
          </AdminLayout>
        </Route>

        <Route path="/admin/catalog">
          <AdminLayout>
            <CatalogList />
          </AdminLayout>
        </Route>

        <Route path="/admin/users">
          <AdminLayout>
            <UserList />
          </AdminLayout>
        </Route>

        <Route path="/admin">
          <Redirect to="/admin/catalog" />
        </Route>

        <Route path="/parent/puzzles/:id">
          <ParentLayout><PuzzleDetail /></ParentLayout>
        </Route>
        <Route path="/parent/puzzles">
          <ParentLayout><PuzzleList /></ParentLayout>
        </Route>
        <Route path="/parent/children">
          <ParentLayout><ChildList /></ParentLayout>
        </Route>
        <Route path="/parent">
          <Redirect to="/parent/puzzles" />
        </Route>

        <Route path="/play/:id" component={GameScreen} />
        <Route path="/reward/:id" component={RewardPage} />

        <Route path="/catalog" component={CatalogPublic} />

        <Route path="/about" component={About} />

        <Route path="/">
          <Redirect to="/catalog" />
        </Route>
      </Switch>
    </I18nProvider>
  );
}
