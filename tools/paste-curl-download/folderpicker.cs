using System;
using System.Runtime.InteropServices;

public static class ExplorerFolderPicker
{
    public static string Pick(string title, string initialPath)
    {
        var dialog = (IFileOpenDialog)new FileOpenDialog();
        try
        {
            uint opts;
            dialog.GetOptions(out opts);
            opts |= FOS_PICKFOLDERS | FOS_FORCEFILESYSTEM | FOS_PATHMUSTEXIST;
            dialog.SetOptions(opts);
            if (!string.IsNullOrEmpty(title))
                dialog.SetTitle(title);
            if (!string.IsNullOrEmpty(initialPath))
            {
                Guid iid = typeof(IShellItem).GUID;
                IShellItem folder;
                if (SHCreateItemFromParsingName(initialPath, IntPtr.Zero, ref iid, out folder) == 0 && folder != null)
                    dialog.SetFolder(folder);
            }
            if (dialog.Show(IntPtr.Zero) != 0)
                return null;
            IShellItem item;
            dialog.GetResult(out item);
            string path;
            item.GetDisplayName(SIGDN_FILESYSPATH, out path);
            return path;
        }
        finally
        {
            Marshal.ReleaseComObject(dialog);
        }
    }

    const uint FOS_PICKFOLDERS = 0x00000020;
    const uint FOS_FORCEFILESYSTEM = 0x00000040;
    const uint FOS_PATHMUSTEXIST = 0x00000800;
    const uint SIGDN_FILESYSPATH = 0x80058000;

    [ComImport, Guid("DC1C5A9C-E88A-4DDE-A5A1-60F82A20AEF7")]
    class FileOpenDialog { }

    [ComImport, Guid("d57c7288-d4ad-4768-be02-9d969532d960"), InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
    interface IFileOpenDialog
    {
        [PreserveSig] int Show(IntPtr parent);
        void SetFileTypes(uint cFileTypes, IntPtr rgFilterSpec);
        void SetFileTypeIndex(uint iFileType);
        void GetFileTypeIndex(out uint piFileType);
        void Advise(IntPtr pfde, out uint pdwCookie);
        void Unadvise(uint dwCookie);
        void SetOptions(uint fos);
        void GetOptions(out uint pfos);
        void SetDefaultFolder(IShellItem psi);
        void SetFolder(IShellItem psi);
        void GetFolder(out IShellItem ppsi);
        void GetCurrentSelection(out IShellItem ppsi);
        void SetFileName([MarshalAs(UnmanagedType.LPWStr)] string pszName);
        void GetFileName([MarshalAs(UnmanagedType.LPWStr)] out string pszName);
        void SetTitle([MarshalAs(UnmanagedType.LPWStr)] string pszTitle);
        void SetOkButtonLabel([MarshalAs(UnmanagedType.LPWStr)] string pszText);
        void SetFileNameLabel([MarshalAs(UnmanagedType.LPWStr)] string pszLabel);
        void GetResult(out IShellItem ppsi);
        void AddPlace(IShellItem psi, uint fdap);
        void SetDefaultExtension([MarshalAs(UnmanagedType.LPWStr)] string pszDefaultExtension);
        void Close(int hr);
        void SetClientGuid(ref Guid guid);
        void ClearClientData();
        void SetFilter(IntPtr pFilter);
        void GetResults(out IntPtr ppenum);
        void GetSelectedItems(out IntPtr ppsai);
    }

    [ComImport, Guid("43826D1E-E718-42EE-BC55-A1E261C37BFE"), InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
    interface IShellItem
    {
        void BindToHandler(IntPtr pbc, ref Guid bhid, ref Guid riid, out IntPtr ppv);
        void GetParent(out IShellItem ppsi);
        void GetDisplayName(uint sigdnName, [MarshalAs(UnmanagedType.LPWStr)] out string ppszName);
        void GetAttributes(uint sfgaoMask, out uint psfgaoAttribs);
        void Compare(IShellItem psi, uint hint, out int piOrder);
    }

    [DllImport("shell32.dll", CharSet = CharSet.Unicode, PreserveSig = true)]
    static extern int SHCreateItemFromParsingName(
        [MarshalAs(UnmanagedType.LPWStr)] string pszPath,
        IntPtr pbc,
        ref Guid riid,
        out IShellItem ppv);
}
