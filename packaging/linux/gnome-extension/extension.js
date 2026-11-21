import Gio from 'gi://Gio';

import * as Main from 'resource:///org/gnome/shell/ui/main.js';
import {Extension} from 'resource:///org/gnome/shell/extensions/extension.js';

const INTERFACE = `
<node>
  <interface name="io.github.utkudarilmaz.monctl.Indicator">
    <method name="ShowBrightness">
      <arg type="d" direction="in" name="level"/>
    </method>
  </interface>
</node>`;

const PATH = '/io/github/utkudarilmaz/monctl/Indicator';

export default class MonctlExtension extends Extension {
    enable() {
        this._icon = Gio.Icon.new_for_string('display-brightness-symbolic');
        this._exported = Gio.DBusExportedObject.wrapJSObject(INTERFACE, this);
        this._exported.export(Gio.DBus.session, PATH);
    }

    disable() {
        this._exported.unexport();
        this._exported = null;
        this._icon = null;
    }

    ShowBrightness(level) {
        if (!Number.isFinite(level))
            return;
        Main.osdWindowManager.show(-1, this._icon, null, Math.min(Math.max(level, 0), 1));
    }
}
